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

package controllers

import (
	"fmt"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/service"
	frametypes "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"

	"github.com/go-logr/logr"
	"github.com/rotisserie/eris"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilerLifeCycle ...
type ReconcilerLifeCycle struct {
	client.Client
	serviceDiscoveryMap map[string]service.ServiceDiscovery
	Logger              logr.Logger
	Registry            frametypes.Registry
}

// UpdateServiceDiscovery ...
func (r *ReconcilerLifeCycle) UpdateServiceDiscovery(obj client.Object) error {
	// if service external discovery type is updated to another type, clean it
	var (
		svc service.Service
	)

	switch obj.GetObjectKind().GroupVersionKind().Kind {
	case "BkGatewayService":
		instance, _ := obj.(*gatewayv1beta1.BkGatewayService)
		svc.TypeMeta = instance.TypeMeta
		svc.ObjectMeta = instance.ObjectMeta
		svc.Upstream = instance.Spec.Upstream
	case "BkGatewayResource":
		instance, _ := obj.(*gatewayv1beta1.BkGatewayResource)
		svc.TypeMeta = instance.TypeMeta
		svc.ObjectMeta = instance.ObjectMeta
		svc.Upstream = instance.Spec.Upstream
	default:
		err := eris.Errorf(
			"Unrecognized type when reconcling with kind %s",
			obj.GetObjectKind().GroupVersionKind().Kind,
		)
		r.Logger.Error(
			err,
			"",
			"kind",
			obj.GetObjectKind().GroupVersionKind(),
			"namespace",
			obj.GetNamespace(),
			"name",
			obj.GetName(),
		)
		return err
	}
	if svc.Upstream == nil || svc.Upstream.ExternalDiscoveryType != r.Registry.Name() {
		r.Logger.V(1).Info("service discovery type do not meet, try to clean it", "obj", obj)
		return r.CleanServiceDiscovery(
			svc.Kind,
			ctrl.Request{NamespacedName: types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}},
		)
	}
	if r.serviceDiscoveryMap == nil {
		r.serviceDiscoveryMap = make(map[string]service.ServiceDiscovery)
	}
	svcKey := r.serviceKey(&svc)
	if _, ok := r.serviceDiscoveryMap[svcKey]; !ok {
		r.Logger.V(1).Info("No service discovery, create One", "obj", svc.ObjectMeta)
		newSD := service.NewServiceDiscovery(r.Client, r.Registry, svcKey)
		r.serviceDiscoveryMap[svcKey] = newSD
	}
	serviceDiscovery := r.serviceDiscoveryMap[svcKey]
	return serviceDiscovery.Apply(&svc)
}

// CleanServiceDiscovery ...
func (r *ReconcilerLifeCycle) CleanServiceDiscovery(kind string, req ctrl.Request) error {
	svcKey := fmt.Sprintf("%s/%s/%s", kind, req.Namespace, req.Name)
	if _, ok := r.serviceDiscoveryMap[svcKey]; !ok {
		r.Logger.V(1).Info("No service discovery, skip clean", "obj", req)
		return nil
	}
	serviceDiscovery := r.serviceDiscoveryMap[svcKey]
	serviceDiscovery.Clean()
	delete(r.serviceDiscoveryMap, svcKey)
	return nil
}

func (r *ReconcilerLifeCycle) serviceKey(svc *service.Service) string {
	return fmt.Sprintf("%s/%s/%s", svc.Kind, svc.Namespace, svc.Name)
}
