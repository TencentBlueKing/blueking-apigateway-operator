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

package leaderelection

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

// EtcdLeaderElector ...
type EtcdLeaderElector struct {
	ctx context.Context

	client     *clientv3.Client
	session    *concurrency.Session
	election   *concurrency.Election
	closeCh    chan struct{}
	leadingCh  chan struct{}
	prefix     string
	instanceID string
	leading    bool
	running    bool

	logger *zap.SugaredLogger
}

// NewEtcdLeaderElector ...
func NewEtcdLeaderElector(client *clientv3.Client, prefix string) (LeaderElector, error) {
	return &EtcdLeaderElector{
		client: client,
		prefix: prefix + "-leader-election",
		instanceID: fmt.Sprintf(
			"%s_%s",
			config.InstanceName,
			config.InstanceIP,
		),
		leading: false,
		running: false,
		logger:  logging.GetLogger().Named("leader-election"),
	}, nil
}

// Run ...
func (ele *EtcdLeaderElector) Run(ctx context.Context) {
	if ele.running {
		return
	}
	ele.ctx = ctx
	ele.running = true
	ele.initElection()
	go ele.run()
}

func (ele *EtcdLeaderElector) initElection() {
	for {
		session, err := concurrency.NewSession(ele.client)
		if err != nil {
			ele.logger.Error(err, "Create election session failed")
			time.Sleep(time.Second * 5)
			continue
		}
		ele.session = session
		break
	}
	ele.election = concurrency.NewElection(ele.session, ele.prefix)
	ele.closeCh = make(chan struct{})
	ele.leadingCh = make(chan struct{})
}

func (ele *EtcdLeaderElector) run() {
	ele.elect()
	ele.checkLeadership()
}

func (ele *EtcdLeaderElector) elect() {
	for {
		ele.logger.Infow("Try to be leader", "id", ele.instanceID)
		err := ele.election.Campaign(ele.ctx, ele.instanceID)
		if err != nil {
			ele.logger.Error(err, "Leader election compaign returns error", "id", ele.instanceID)
			time.Sleep(time.Second * 5)
			continue
		}
		ele.logger.Infow("Become leader now", "id", ele.instanceID)
		ele.leading = true
		close(ele.leadingCh)
		return
	}
}

func (ele *EtcdLeaderElector) checkLeadership() {
	for {
		select {
		case <-ele.session.Done():
			close(ele.closeCh)
			ele.leading = false
			ele.initElection()
			go ele.run()
			return
		case <-ele.ctx.Done():
			ele.session.Close()
			close(ele.closeCh)
			ele.leading = false
			ele.running = false
			return
		}
	}
}

// Leader ...
func (ele *EtcdLeaderElector) Leader() string {
	if ele.election == nil {
		return ""
	}
	resp, err := ele.election.Leader(ele.ctx)
	if err != nil {
		ele.logger.Error(err, "Get Leader info failed")
		return ""
	}
	if resp.Count == 0 {
		return ""
	}
	return string(resp.Kvs[0].Value)
}

// WaitForLeading ...
func (ele *EtcdLeaderElector) WaitForLeading() (closeCh <-chan struct{}) {
	if ele.leading {
		return ele.closeCh
	}
	<-ele.leadingCh
	return ele.closeCh
}
