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

// Package runner ...
package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/agent"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/committer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/store"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/server"
)

// EtcdAgentRunner ...
type EtcdAgentRunner struct {
	client            *clientv3.Client
	apigwEtcdRegistry *registry.APIGWEtcdRegistry
	leader            *leaderelection.EtcdLeaderElector
	synchronizer      *synchronizer.ApisixConfigSynchronizer
	apisixEtcdstore   *store.ApisixEtcdStore

	committer *committer.Committer
	agent     *agent.EventAgent

	cfg *config.Config

	logger *zap.SugaredLogger
}

// NewEtcdAgentRunner ...
func NewEtcdAgentRunner(cfg *config.Config) *EtcdAgentRunner {
	client, err := initOperatorEtcdClient(cfg)
	if err != nil {
		fmt.Println(err, "Error creating apigwEtcdRegistry etcd client")
		os.Exit(1)
	}

	r := &EtcdAgentRunner{
		client: client,
		cfg:    cfg,
		logger: logging.GetLogger().Named("etcd-agent-runner"),
	}
	r.init()
	return r
}

func (r *EtcdAgentRunner) init() {
	// 1. init metrics
	metric.InitMetric(prometheus.DefaultRegisterer)

	// 2. init apigwEtcdRegistry
	r.apigwEtcdRegistry = registry.NewAPIGWEtcdRegistry(r.client, r.cfg.Dashboard.Etcd.KeyPrefix)

	// 3. init leader elector
	if r.cfg.Operator.WithLeader {
		r.leader, _ = leaderelection.NewEtcdLeaderElector(r.client, r.cfg.Dashboard.Etcd.KeyPrefix)
	}

	// 4. init output
	apisixEtcdStore, err := initApisixEtcdStore(r.cfg)
	if err != nil {
		fmt.Println(err, "Error creating etcd apisixEtcdstore")
		os.Exit(1)
	}
	r.apisixEtcdstore = apisixEtcdStore
	r.synchronizer = synchronizer.NewSynchronizer(apisixEtcdStore, "/healthz")

	stageTimer := timer.NewReleaseTimer()
	// 5. init committer
	r.committer = committer.NewCommitter(
		r.apigwEtcdRegistry,
		r.synchronizer,
		stageTimer,
	)
	commitChan := r.committer.GetCommitChan()

	// 6. init agent
	r.agent = agent.NewEventAgent(
		r.apigwEtcdRegistry,
		commitChan,
		r.synchronizer,
		stageTimer,
	)
}

// Run ...
func (r *EtcdAgentRunner) Run(ctx context.Context) {
	// 1. run http server
	httpServer := server.NewServer(
		r.leader,
		r.apigwEtcdRegistry,
		r.apisixEtcdstore,
		r.committer,
	)
	httpServer.RegisterMetric(prometheus.DefaultGatherer)
	httpServer.Run(ctx, r.cfg)

	// 2. waiting leader election
	var keepAliveChan <-chan struct{} = make(chan struct{})
	if r.leader != nil {
		r.leader.Run(ctx)
		r.logger.Info("waiting for becoming leader...")
		keepAliveChan = r.leader.WaitForLeading()
	}

	// 3. run committer
	r.logger.Info("starting committer")
	go r.committer.Run(ctx)

	// 4. run agent
	r.agent.SetKeepAliveChan(keepAliveChan)

	r.logger.Info("starting etcd agent")
	r.agent.Run(ctx)
	r.logger.Error("Agent stopped running")
}
