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
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/token"
)

// BkGatewayConfigReconciler reconciles a BkGatewayConfig object
type BkGatewayConfigReconciler struct {
	Handler registry.KubeEventHandler
	client.Client
	Scheme *runtime.Scheme
	Issuer *token.Issuer
}

//nolint:lll
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BkGatewayConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *BkGatewayConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// reportInterval := time.Duration(viper.GetInt(constant.FlagKeyBkGatewayStatusReportInterval)) * time.Second
	reportInterval := 30 * time.Second

	config := &gatewayv1beta1.BkGatewayConfig{}
	if err := r.Get(ctx, k8stypes.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, config); err != nil {
		logger.Error(err, fmt.Sprintf("get bk gateway config %s/%s failed", req.Name, req.Namespace))
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: reportInterval,
		}, nil
	}
	r.Handler.KubeEventHandler(&registry.ResourceMetadata{})

	// TODO::反向同步相关逻辑，目前链路未打通，先屏蔽，后续完善
	// if err := r.reportStatus(ctx, config); err != nil {
	// 	logger.Error(err, "report status to edge controller failed")
	// 	return ctrl.Result{
	// 		Requeue:      true,
	// 		RequeueAfter: reportInterval,
	// 	}, nil
	// }

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: reportInterval,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BkGatewayConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1beta1.BkGatewayConfig{}).
		Watches(&source.Kind{Type: &gatewayv1beta1.BkGatewayInstance{}}, &handler.Funcs{}).
		Watches(&source.Kind{Type: &gatewayv1beta1.BkGatewayStage{}}, &handler.Funcs{}).
		Complete(r)
}
