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

package discoveryoperatorframe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rotisserie/eris"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gatewayv1beta1 "micro-gateway/api/v1beta1"
	"micro-gateway/pkg/discovery-operator-frame/controllers"
	"micro-gateway/pkg/discovery-operator-frame/options"
	"micro-gateway/pkg/discovery-operator-frame/types"
	"micro-gateway/pkg/discovery-operator-frame/utils"
)

// DiscoveryOperator is interface for frame operator
type DiscoveryOperator interface {
	Run() error
	GetKubeClient() client.Client
}

// DiscoveryOperatorFrame is implementation for DiscoveryOperator
type DiscoveryOperatorFrame struct {
	opts   options.FrameOptions
	mgr    manager.Manager
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(gatewayv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// NewDiscoveryOperator create new discovery operator frame
func NewDiscoveryOperator(opts options.FrameOptions) (DiscoveryOperator, error) {
	setupLog = setupLog.WithValues("registry", opts.Registry.Name())
	// validate registry name
	if !utils.SubDomainCheck(opts.Registry.Name()) {
		setupLog.Error(
			nil,
			"registry's name does not meet subdomain's requirements",
			"subdomainRegex",
			utils.SubDomainRegex,
		)
		return nil, eris.Errorf("registry's name does not meet subdomain's requirements")
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts.ZapOpts)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      opts.MetricsAddress,
		Port:                    9443,
		Namespace:               opts.RegisterNamespace,
		HealthProbeBindAddress:  opts.HealthProbeBindAddress,
		LeaderElection:          opts.LeaderElection,
		LeaderElectionNamespace: opts.RegisterNamespace,
		LeaderElectionID:        fmt.Sprintf("%s.bk.tencent.com", opts.Registry.Name()),
	})
	if err != nil {
		setupLog.Error(err, "unable to start disovery operator frame")
		return nil, err
	}

	// setup reconciler lifecycle controller
	lifecycle := &controllers.ReconcilerLifeCycle{
		Client:   mgr.GetClient(),
		Logger:   ctrl.Log.WithName("DiscoveryOperator").WithValues("registryName", opts.Registry.Name()),
		Registry: opts.Registry,
	}
	// setup BkGatewayService reconciler
	if err = (&controllers.BkGatewayServiceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		LifeCycle: lifecycle,
		Registry:  opts.Registry,
		Namespace: opts.RegisterNamespace,
		Logger:    ctrl.Log.WithName("BkGatewayServiceReconciler").WithValues("registryName", opts.Registry.Name()),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BkGatewayServiceReconciler")
		return nil, err
	}
	// setup BkGatewayResource reconciler
	if err = (&controllers.BkGatewayResourceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		LifeCycle: lifecycle,
		Registry:  opts.Registry,
		Namespace: opts.RegisterNamespace,
		Logger:    ctrl.Log.WithName("BkGatewayResourceReconciler").WithValues("registryName", opts.Registry.Name()),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BkGatewayResourceReconciler")
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &DiscoveryOperatorFrame{opts: opts, mgr: mgr, ctx: ctx, cancel: cancel}, nil
}

// Run run the operator frame. Run function will block, and is not thread safe when
// leader election is disabled
func (d *DiscoveryOperatorFrame) Run() error {
	// startup cleanup
	go d.cleanUpOutdatedEndpoints()
	// register gateway operator crd
	go d.periodlyRegisterGatewayOperator()
	setupLog.Info("starting manager")
	// start manager
	if err := d.mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		d.cancel()
		setupLog.Error(err, "problem running manager")
		return err
	}
	d.cancel()
	setupLog.Info("manager stopped without error")
	return nil
}

// GetKubeClient get k8s client
func (d *DiscoveryOperatorFrame) GetKubeClient() client.Client {
	return d.mgr.GetClient()
}

func (d *DiscoveryOperatorFrame) cleanUpOutdatedEndpoints() error {
	<-d.mgr.Elected()
	setupLog.Info("cleanup outdated endpoints")

	// list endpoints which managed by discovery itself
	epsList := &gatewayv1beta1.BkGatewayEndpointsList{}
	cli := d.mgr.GetClient()
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", types.ManagedByLabelTag, d.opts.Registry.Name()))
	if err != nil {
		setupLog.Error(err, "Build selector for list BkGatewayEndpoints failed")
		return err
	}
	err = cli.List(context.Background(), epsList, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     d.opts.RegisterNamespace,
	})
	if err != nil {
		setupLog.Error(err, "List BkGatewayEndpoints failed")
		return err
	}
	// check whether endpoints corresponding service has changed config type or been deleted
	for _, eps := range epsList.Items {
		namelist := strings.Split(eps.Name, ".")
		if len(namelist) != 3 {
			setupLog.Error(
				nil,
				"Split endpoints name to service name and registry name failed, skip startup checking for this endpoints",
				"seperator",
				types.EndpointsNameSeperator,
				"endpointsName",
				eps.Name,
			)
			continue
		}
		if d.opts.Registry.Name() != namelist[0] {
			setupLog.Error(
				nil,
				"Endpoints name does not equals to registry name, skip startup checking for this endpoints",
				"registryName",
				d.opts.Registry.Name(),
				"endpointsNameIndex0",
				namelist[0],
			)
			continue
		}
		factory, ok := types.AbbObjectFactoryMapping[namelist[1]]
		if !ok {
			setupLog.Error(
				nil,
				"resource type does not support, skip startup checking for this endpoints",
				"registryName",
				d.opts.Registry.Name(),
				"endpointsName",
				eps.Name,
			)
			continue
		}
		obj := factory()
		err := cli.Get(context.Background(), client.ObjectKey{Namespace: eps.Namespace, Name: namelist[2]}, obj)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				setupLog.Info(
					"BkGatewayService not found, clean endpoints",
					"service",
					fmt.Sprintf("%s/%s", eps.Namespace, namelist[1]),
					"endpoints",
					fmt.Sprintf("%s/%s", eps.Namespace, eps.Name),
				)
				innerErr := cli.Delete(context.Background(), &eps)
				if innerErr != nil {
					setupLog.Error(
						innerErr,
						"Delete BkGatewayService failed",
						"endpoints",
						fmt.Sprintf("%s/%s", eps.Namespace, eps.Name),
					)
					continue
				}
			} else {
				setupLog.Error(
					err,
					"Get BkGatewayService failed",
					"service",
					fmt.Sprintf("%s/%s", eps.Namespace, namelist[1]),
					"endpoints",
					fmt.Sprintf("%s/%s", eps.Namespace, eps.Name),
				)
				continue
			}
		}
		discoveryType := ""
		switch v := obj.(type) {
		case *gatewayv1beta1.BkGatewayService:
			if v.Spec.Upstream != nil {
				discoveryType = v.Spec.Upstream.ExternalDiscoveryType
			}
		case *gatewayv1beta1.BkGatewayResource:
			if v.Spec.Upstream != nil {
				discoveryType = v.Spec.Upstream.ExternalDiscoveryType
			}
		}
		if discoveryType != d.opts.Registry.Name() {
			setupLog.Info(
				"BkGatewayService do not use this registry, clean endpoints",
				"service",
				fmt.Sprintf("%s/%s", eps.Namespace, namelist[1]),
				"endpoints",
				fmt.Sprintf("%s/%s", eps.Namespace, eps.Name),
				"configType",
				discoveryType,
			)
			err = cli.Delete(context.Background(), &eps)
			if err != nil {
				setupLog.Error(
					err,
					"Delete BkGatewayService failed",
					"endpoints",
					fmt.Sprintf("%s/%s", eps.Namespace, eps.Name),
				)
				continue
			}
		}
	}
	return nil
}

func (d *DiscoveryOperatorFrame) periodlyRegisterGatewayOperator() error {
	<-d.mgr.Elected()
	setupLog.Info("periodly register gateway operator")
	d.registerGatewayOperatorOnce()

	// register BkGatewayOperator CR
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := d.registerGatewayOperatorOnce()
			if err != nil {
				setupLog.Error(err, "")
			}
		case <-d.ctx.Done():
			d.deRegisterGatewayOperator()
			return nil
		}
	}
}

func (d *DiscoveryOperatorFrame) registerGatewayOperatorOnce() error {
	expireTime := func() metav1.Time {
		return metav1.NewTime(time.Now().Add(time.Minute * 5))
	}

	operator := &gatewayv1beta1.BkGatewayOperator{}
	operator.SetNamespace(d.opts.RegisterNamespace)
	operator.SetName(d.opts.Registry.Name())
	rawSchema, err := utils.Map2RawExtension(d.opts.ConfigSchema)
	if err != nil {
		setupLog.Error(err, "", "schema", d.opts.ConfigSchema)
		rawSchema = runtime.RawExtension{}
	}

	// GetOrCreate BkGatewayOperator
	cli := d.mgr.GetClient()
	err = cli.Get(context.Background(), client.ObjectKeyFromObject(operator), operator)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			operator.Spec.DiscoveryType = d.opts.Registry.Name()
			operator.Spec.ConfigSchema = rawSchema
			operator.Status.Status = gatewayv1beta1.BkGatewayOperatorStatusReady
			operator.Status.ReadyUntil = expireTime()
			err = cli.Create(context.Background(), operator)
			if err != nil {
				setupLog.Error(err, "Register BkGatewayOperator failed")
				return err
			}
		} else {
			setupLog.Error(err, fmt.Sprintf("Get BkGatewayOperator failed: %v", err))
			return err
		}
	}
	// Update BkGatewayOperator
	operator.Spec.DiscoveryType = d.opts.Registry.Name()
	operator.Spec.ConfigSchema = rawSchema
	err = cli.Update(context.Background(), operator)
	if err != nil {
		setupLog.Error(err, "Register BkGatewayOperator failed, update failed")
		return err
	}
	operator.Status.Status = gatewayv1beta1.BkGatewayOperatorStatusReady
	operator.Status.ReadyUntil = expireTime()
	err = cli.Status().Update(context.Background(), operator)
	if err != nil {
		setupLog.Error(err, "Register BkGatewayOperator failed, update status failed")
		return err
	}
	return nil
}

// deRegisterGatewayOperator delete BkGatewayOperator CR
func (d *DiscoveryOperatorFrame) deRegisterGatewayOperator() {
	operator := &gatewayv1beta1.BkGatewayOperator{}
	operator.SetNamespace(d.opts.RegisterNamespace)
	operator.SetName(d.opts.Registry.Name())

	cli := d.mgr.GetClient()
	err := cli.Get(context.Background(), client.ObjectKeyFromObject(operator), operator)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			setupLog.Info("BkGatewayOperator register record does not exist, skip deleting")
			return
		}
		setupLog.Error(err, "BkGatewayOperator deRegister failed, get failed")
		return
	}
	err = cli.Delete(context.Background(), operator)
	if err != nil {
		setupLog.Error(err, "BkGatewayOperator deRegister failed, delete failed")
	}
}
