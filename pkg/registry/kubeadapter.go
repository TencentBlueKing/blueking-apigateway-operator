/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package registry

import (
	"context"
	"reflect"
	"sync"

	uuid "github.com/satori/go.uuid"
	"github.com/smallnest/chanx"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"micro-gateway/api/v1beta1"
	"micro-gateway/pkg/config"
	"micro-gateway/pkg/logging"
)

// KubeEventHandler is the interface that defines how to handle a Resource Metadata
// object, which is derived from Kubernetes events.
type KubeEventHandler interface {
	KubeEventHandler(*ResourceMetadata)
}

// K8SRegistryAdapter implements the Register interface using Kubernetes apiserver
// as the main storage.
type K8SRegistryAdapter struct {
	kubeClient client.Client
	namespace  string

	watchChMap sync.Map

	logger *zap.SugaredLogger
}

// NewK8SResourceRegistry creates a Registry and a KubeEventHandler object.
//
// NOTE: Both return values are identical (a `K8SRegistryAdapter` object) while they
// implement different interfaces(Registry, KubeEventHandler); maybe we should divide
// the `KubeEventHandler` into another type in the future.
func NewK8SResourceRegistry(kubeClient client.Client, namespace string) (*K8SRegistryAdapter, *K8SRegistryAdapter) {
	registry := &K8SRegistryAdapter{
		kubeClient: kubeClient,
		namespace:  namespace,
		logger:     logging.GetLogger().Named("k8s-resource-registry"),
	}
	return registry, registry
}

// Get a resource by its resource key, key's ResourceName must not be empty and the
// object must be a pointer to a struct so that it can be updated.
func (r *K8SRegistryAdapter) Get(ctx context.Context, key ResourceKey, obj client.Object) error {
	return r.kubeClient.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: key.ResourceName}, obj)
}

// ListStages returns a slice of StageInfo objects under current namespace.
func (r *K8SRegistryAdapter) ListStages(ctx context.Context) ([]StageInfo, error) {
	list := &v1beta1.BkGatewayStageList{}
	err := r.kubeClient.List(ctx, list, &client.ListOptions{Namespace: r.namespace})
	if err != nil {
		return nil, err
	}
	stageInfoList := make([]StageInfo, 0)
	for _, stage := range list.Items {
		si, ok := r.parseStageInfo(&stage)
		if !ok {
			continue
		}
		stageInfoList = append(stageInfoList, si)
	}
	return stageInfoList, nil
}

// List a slice of objects by a given resource key.
func (r *K8SRegistryAdapter) List(ctx context.Context, key ResourceKey, obj client.ObjectList) error {
	labelSelector := labels.NewSelector()
	if key.GatewayName != "" {
		sr, _ := labels.NewRequirement(
			config.BKAPIGatewayLabelKeyGatewayName,
			selection.Equals,
			[]string{key.GatewayName},
		)
		labelSelector = labelSelector.Add(*sr)
	}
	if key.StageName != "" {
		sr, _ := labels.NewRequirement(
			config.BKAPIGatewayLabelKeyGatewayStage,
			selection.Equals,
			[]string{key.StageName},
		)
		labelSelector = labelSelector.Add(*sr)
	}
	if key.ResourceName != "" {
		sr, _ := labels.NewRequirement(
			config.BKAPIGatewayLabelKeyResourceName,
			selection.Equals,
			[]string{key.ResourceName},
		)
		labelSelector = labelSelector.Add(*sr)
	}
	return r.kubeClient.List(ctx, obj, &client.ListOptions{Namespace: r.namespace, LabelSelector: labelSelector})
}

// Watch creates and returns a channel that produces update events of resources.
func (r *K8SRegistryAdapter) Watch(ctx context.Context) <-chan *ResourceMetadata {
	retCh := make(chan *ResourceMetadata)
	id := uuid.NewV1()
	// 100 is only the initial capacity, chanx will extend the buffer size when it is full.
	ubc := chanx.NewUnboundedChan(100)
	r.watchChMap.Store(id, ubc)
	go func() {
		defer close(retCh)
		defer close(ubc.In)
		for {
			select {
			case <-ctx.Done():
				r.watchChMap.Delete(id)
				return
			case v := <-ubc.Out:
				retCh <- v.(*ResourceMetadata)
			}
		}
	}()
	return retCh
}

// KubeEventHandler pushes a ResourceMetadata object to all bound channels.
func (r *K8SRegistryAdapter) KubeEventHandler(rm *ResourceMetadata) {
	r.logger.Debugw("Resource event triggered", "resourceMeta", rm)
	r.watchChMap.Range(func(key, value interface{}) bool {
		ubc := value.(*chanx.UnboundedChan)
		ubc.In <- rm
		return true
	})
}

// Parse StageInfo obj from stage object.
//
// Return the info object and a Boolean value which indicates if a non-empty value exists.
func (r *K8SRegistryAdapter) parseStageInfo(obj client.Object) (StageInfo, bool) {
	labels := obj.GetLabels()
	// NOTE: Are there any chances that the key exists but the value is empty?
	gateway, okGw := labels[config.BKAPIGatewayLabelKeyGatewayName]
	stage, okStg := labels[config.BKAPIGatewayLabelKeyGatewayStage]
	if !(okGw && okStg) {
		r.logger.Error(nil, "object has no associated gateway and stage.",
			"labels", obj.GetLabels(),
			"name", obj.GetName,
			"namespace", obj.GetNamespace,
			"type", reflect.TypeOf(obj).Name())
		return emptyStageInfo, false
	}
	return StageInfo{GatewayName: gateway, StageName: stage}, true
}
