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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// BkGatewayTLSControlelr reconciles a BkGatewayTLS object
type BkGatewayTLSControlelr struct {
	adapater *registryAdapter
	Handler  registry.KubeEventHandler
	client.Client
	Scheme *runtime.Scheme
}

//nolint:lll
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BkGatewayResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *BkGatewayTLSControlelr) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("secret trigger", "obj", req)
	r.adapater.Reconcile(ctx, req, &gatewayv1beta1.BkGatewayTLS{}, logger)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BkGatewayTLSControlelr) SetupWithManager(mgr ctrl.Manager) error {
	r.adapater = &registryAdapter{
		resMetaMap: make(map[types.NamespacedName]*registry.ResourceMetadata),
		handler:    r.Handler,
		Client:     r.Client,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1beta1.BkGatewayTLS{}).
		Complete(r)
}
