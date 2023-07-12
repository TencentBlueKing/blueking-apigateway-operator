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

package runner

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/controllers"
	"github.com/TencentBlueKing/blueking-apigateway-operator/internal/token"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/server"
)

var (
	scheme                = runtime.NewScheme()
	DefaultHealthzHandler *healthz.Handler
)

// KubeAgentRunner ...
type KubeAgentRunner struct {
	client       client.Client
	manager      manager.Manager
	registry     registry.Registry
	leader       leaderelection.LeaderElector
	synchronizer *synchronizer.ApisixConfigurationSynchronizer
	store        synchronizer.ApisixConfigStore

	commiter *commiter.Commiter
	agent    *agent.EventAgent

	cfg    *config.Config
	logger *zap.SugaredLogger
}

// NewKubeAgentRunner ...
func NewKubeAgentRunner(cfg *config.Config) *KubeAgentRunner {
	r := &KubeAgentRunner{
		cfg:    cfg,
		logger: logging.StdoutLogger().Named("kube-agent-runner"),
	}
	r.init()
	return r
}

func (r *KubeAgentRunner) preInit() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gatewayv1beta1.AddToScheme(scheme))

	DefaultHealthzHandler = &healthz.Handler{
		Checks: make(map[string]healthz.Checker),
	}
	DefaultHealthzHandler.Checks["healthz"] = healthz.Ping

	//+kubebuilder:scaffold:scheme
}

func (r *KubeAgentRunner) init() {
	// 1. pre init
	r.preInit()

	// 1. init metrics
	metric.InitMetric(metrics.Registry)

	// 2. init k8s client
	r.manager = r.initManager()
	r.client = r.manager.GetClient()
	r.logger.Info("starting manager")

	// 3. init registry
	r.initRegistry()

	// 4. init leader elector
	r.initLeaderElector()

	// 5. init output
	store, err := initApisixConfigStore(r.cfg)
	if err != nil {
		r.logger.Error(err, "Error creating etcd store")
		os.Exit(1)
	}
	r.store = store
	r.synchronizer = synchronizer.NewSynchronizer(store, "/healthz")

	// 6. init commiter
	radixTreeGetter := radixtree.NewSingleRadixTreeGetter()
	stageTimer := timer.NewStageTimer()

	r.commiter = commiter.NewCommiter(
		r.registry,
		r.synchronizer,
		radixTreeGetter,
		stageTimer,
		r.client,
	)
	commitChan := r.commiter.GetCommitChan()

	// 6. init agent
	r.agent = agent.NewEventAgent(
		r.registry,
		commitChan,
		r.synchronizer,
		radixTreeGetter,
		stageTimer,
	)
}

func (r *KubeAgentRunner) initLeaderElector() {
	if r.cfg.Operator.WithLeader {
		electionNameWithApisixPrefix := r.cfg.KubeExtension.LeaderElectionName
		// constant.FlagKeyApisixEtcdKeyPrefix => "/bk-gateway-apisix-v1"
		// []string 0:""  1:"bk-gateway-apisix-v1"
		apisixPrefixSplits := strings.Split(r.cfg.Apisix.Etcd.KeyPrefix, "/")
		// avoid of array out of bounds panic
		if len(apisixPrefixSplits) >= 2 {
			electionNameWithApisixPrefix += "-" + apisixPrefixSplits[1]
		}
		leader, err := leaderelection.NewKubeLeaderElector(r.cfg.KubeExtension.LeaderElectionType,
			electionNameWithApisixPrefix,
			config.InstanceNamespace,
			"",
			time.Duration(r.cfg.KubeExtension.LeaderElectionLeaseDuration)*time.Second,
			time.Duration(r.cfg.KubeExtension.LeaderElectionRenewDuration)*time.Second,
			time.Duration(r.cfg.KubeExtension.LeaderElectionRetryDuration)*time.Second,
		)
		if err != nil {
			r.logger.Error(err, "create leader election failed")
			os.Exit(1)
		}

		r.leader = leader
	}
}

func (r *KubeAgentRunner) initRegistry() {
	if r.cfg.Operator.AgentMode {
		// use etcd register
		client, err := initOperatorEtcdClient(r.cfg)
		if err != nil {
			r.logger.Error(nil, "Error creating etcd client")
			os.Exit(1)
		}
		r.registry = registry.NewEtcdResourceRegistry(client, r.cfg.Dashboard.Etcd.KeyPrefix)
	} else {
		// use kube register
		var handler registry.KubeEventHandler
		r.registry, handler = registry.NewK8SResourceRegistry(r.client, r.cfg.KubeExtension.WorkNamespace)
		issuer := r.initIssuer()
		r.registerController(r.manager, issuer, handler)
	}
}

// Run ...
func (r *KubeAgentRunner) Run(ctx context.Context) {
	// 1. run http server
	server := server.NewServer(
		r.leader,
		r.registry,
		r.store,
		r.commiter,
	)
	server.RegisterMetric(metrics.Registry)
	server.Run(ctx, r.cfg)

	// 2. start k8s manager
	go func() {
		if err := r.manager.Start(ctrl.SetupSignalHandler()); err != nil {
			r.logger.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()

	// 3. waiting leader election
	var keepAliveChan <-chan struct{} = make(chan struct{})
	if r.leader != nil {
		r.leader.Run(ctx)
		r.logger.Info("waiting for becoming leader...")
		keepAliveChan = r.leader.WaitForLeading()
	}

	// 4. wait for k8s cache sync
	if !r.manager.GetCache().WaitForCacheSync(ctx) {
		r.logger.Error(nil, "WaitForCacheSync failed")
		return
	}

	// 5. run commiter
	r.logger.Info("starting commiter")
	go r.commiter.Run(ctx)

	// 6. run agent
	r.agent.SetKeepAliveChan(keepAliveChan)

	r.logger.Info("starting kube agent")
	r.agent.Run(ctx)
	r.logger.Error(nil, "Agent stopped running")
}

func (r *KubeAgentRunner) registerController(
	mgr manager.Manager,
	issuer *token.Issuer,
	handler registry.KubeEventHandler,
) {
	var err error

	if err = (&controllers.SecretController{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "SecretContsoller")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayStageReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayStage")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayServiceReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayService")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayResourceReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayResource")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayConfigReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Issuer:  issuer,
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayConfig")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayInstanceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayInstance")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayPluginMetadataReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayPluginMetadata")
		os.Exit(1)
	}
	if err = (&controllers.BkGatewayTLSControlelr{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Handler: handler,
	}).SetupWithManager(mgr); err != nil {
		r.logger.Error(err, "unable to create controller", "controller", "BkGatewayTLS")
		os.Exit(1)
	}
}

func (r *KubeAgentRunner) initManager() manager.Manager {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		Port:               9443,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          r.cfg.KubeExtension.WorkNamespace,
	})
	if err != nil {
		r.logger.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		r.logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		r.logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	return mgr
}

func (r *KubeAgentRunner) initIssuer() *token.Issuer {
	issuer := token.New("micro-gateway", "", config.InstanceName)
	go issuer.RefreshLoop()
	return issuer
}
