/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2025 Tencent. All rights reserved.
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

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

var emptyRequest = ctrl.Request{} // represents for full synchronization
// emptyResourceMeta = &registry.ResourceMetadata{}

// TODO:: use controller-manager predictor to replace registryAdapter
type registryAdapter struct {
	resMetaMap map[types.NamespacedName]*registry.ResourceMetadata
	handler    registry.KubeEventHandler
	client.Client
}

// Init :
func (ra *registryAdapter) Init() {
	ra.resMetaMap = make(map[types.NamespacedName]*registry.ResourceMetadata)
}

// Reconcile :
func (ra *registryAdapter) Reconcile(
	ctx context.Context,
	req ctrl.Request,
	obj client.Object,
	logger logr.Logger,
) error {
	if req == emptyRequest {
		// ra.handler.KubeEventHandler(emptyResourceMeta)
		return nil
	}
	err := ra.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			rm, ok := ra.resMetaMap[req.NamespacedName]
			if !ok {
				logger.Error(
					err,
					"PluginMetadata seems to be deleted, but no resource metadata saved",
					"NamespacedName",
					req.NamespacedName,
				)
				return nil
			}
			ra.handler.KubeEventHandler(rm)
			delete(ra.resMetaMap, req.NamespacedName)
			return nil
		}
		logger.Error(err, "Get PluginMetadata failed", "NamespacedName", req.NamespacedName)
		return err
	}

	rm, ok := ra.resMetaMap[req.NamespacedName]
	changed := false
	if ok {
		if !isSameStage(rm.StageInfo, obj.GetLabels()) {
			logger.V(1).Info("Resource stage has changed", "old", rm, "new", obj.GetLabels())
			ra.handler.KubeEventHandler(rm)
			changed = true
		}
	}
	// created or stage changes
	if !ok || changed {
		gvk, ok := gatewayv1beta1.GetGVK(obj)
		if !ok {
			logger.Error(nil, "No gvk for provided resource", "type", reflect.TypeOf(obj).Name())
			return nil
		}
		rm = buildResourceMetadata(req.Name, obj.GetLabels(), gvk)
		if rm == nil {
			logger.Info(
				fmt.Sprintf("Resource without labels \"%s\" or \"%s\", will be omitted",
					config.BKAPIGatewayLabelKeyGatewayName,
					config.BKAPIGatewayLabelKeyGatewayStage),
				"req", req)
			delete(ra.resMetaMap, req.NamespacedName)
			return nil
		}
		ra.resMetaMap[req.NamespacedName] = rm
	}
	ra.handler.KubeEventHandler(rm)
	return nil
}

func buildResourceMetadata(
	name string,
	labels map[string]string,
	gvk schema.GroupVersionKind,
) *registry.ResourceMetadata {
	rm := &registry.ResourceMetadata{
		APIVersion: gvk.Version,
		Kind:       gvk.Kind,
		Name:       name,
	}
	// TODO:: make secret resource support stage-scoped
	if gvk.Kind == "Secret" {
		return rm
	}
	var ok bool
	if rm.GatewayName, ok = labels[config.BKAPIGatewayLabelKeyGatewayName]; !ok {
		return nil
	}
	if rm.StageName, ok = labels[config.BKAPIGatewayLabelKeyGatewayStage]; !ok {
		return nil
	}
	rm.Ctx = context.Background()
	return rm
}

func isSameStage(si registry.StageInfo, labels map[string]string) bool {
	return labels[config.BKAPIGatewayLabelKeyGatewayName] == si.GatewayName &&
		labels[config.BKAPIGatewayLabelKeyGatewayStage] == si.StageName
}
