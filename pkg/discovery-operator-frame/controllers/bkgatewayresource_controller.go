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
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"
)

// BkGatewayResourceReconciler reconciles a BkGatewayResource object
type BkGatewayResourceReconciler struct {
	client.Client
	Logger    logr.Logger
	Scheme    *runtime.Scheme
	LifeCycle *ReconcilerLifeCycle
	Registry  types.Registry
	Namespace string
}

//nolint:lll
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.bk.tencent.com,resources=bkgatewayservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BkGatewayResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *BkGatewayResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger.V(1).Info("bk gateway service trigger", "obj", req)
	defer r.Logger.V(1).Info("bk gateway service reconcile finished", "obj", req)
	svc := &gatewayv1beta1.BkGatewayResource{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, svc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// service has been deleted, clean it
			innerErr := r.LifeCycle.CleanServiceDiscovery("BkGatewayResource", req)
			if innerErr != nil {
				r.Logger.Error(innerErr, "Clean service discovery failed.", "BkGatewayResource", req)
				return ctrl.Result{
					RequeueAfter: time.Second * 5,
				}, innerErr
			}
			r.Logger.Info("Clean service discovery succ", "obj", req)
			return ctrl.Result{}, nil
		}
		r.Logger.Error(err, "Get BkGatewayResource from apiserver failed", "obj", req)
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}
	// update service discovery
	r.LifeCycle.UpdateServiceDiscovery(svc)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BkGatewayResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1beta1.BkGatewayResource{}).
		WithEventFilter(r.servicePredicate()).
		Complete(r)
}

// getPolarisConfigPredicate filter PolarisConfig events
func (r *BkGatewayResourceReconciler) servicePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Object.GetNamespace() != r.Namespace {
				return false
			}
			conf, ok := e.Object.(*gatewayv1beta1.BkGatewayResource)
			if !ok {
				return false
			}
			// service external discovery type matched
			if conf.Spec.Upstream != nil && conf.Spec.Upstream.ExternalDiscoveryType == r.Registry.Name() {
				return true
			}
			r.Logger.V(1).
				Info("New BkGatewayResource does not associate with this discovery operator, skip", "svc",
					conf.GetNamespace()+"/"+conf.GetName(), "upstreamConfig", conf.Spec.Upstream)
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetNamespace() != r.Namespace {
				return false
			}
			newconf, okNew := e.ObjectNew.(*gatewayv1beta1.BkGatewayResource)
			oldconf, okOld := e.ObjectOld.(*gatewayv1beta1.BkGatewayResource)
			if !okNew || !okOld {
				return false
			}
			// old or new service external discovery type matched
			if (newconf.Spec.Upstream != nil && newconf.Spec.Upstream.ExternalDiscoveryType == r.Registry.Name()) ||
				(oldconf.Spec.Upstream != nil && oldconf.Spec.Upstream.ExternalDiscoveryType == r.Registry.Name()) {
				return true
			}
			r.Logger.V(1).
				Info(
					"BkGatewayResource does not associate with this discovery operator or spec does not changed, skip",
					"svc", newconf.GetNamespace()+"/"+newconf.GetName(), "upstreamConfig", newconf.Spec.Upstream,
				)
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if e.Object.GetNamespace() != r.Namespace {
				return false
			}
			conf, ok := e.Object.(*gatewayv1beta1.BkGatewayResource)
			if !ok {
				return false
			}
			// service external discovery type matched
			if conf.Spec.Upstream != nil && conf.Spec.Upstream.ExternalDiscoveryType == r.Registry.Name() {
				return true
			}
			r.Logger.V(1).
				Info("Deleted BkGatewayResource does not associate with this discovery operator, skip",
					"svc", conf.GetNamespace()+"/"+conf.GetName(), "upstreamConfig", conf.Spec.Upstream)
			return false
		},
	}
}
