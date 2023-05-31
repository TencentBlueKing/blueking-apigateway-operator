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
	"os"
	"time"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

// KubeLeaderElector client for leader election
type KubeLeaderElector struct {
	ctx context.Context
	// lock type in kubernetes, available [resourcelock.EndpointsResourceLock, resourcelock.LeasesResourceLock ..... ]
	lockType      string
	name          string
	namespace     string
	leaseDuration time.Duration
	renewDuration time.Duration
	retryPeriod   time.Duration
	closeCh       chan struct{}
	leadingCh     chan struct{}

	lock    resourcelock.Interface
	elector *leaderelection.LeaderElector

	isMaster bool

	logger *zap.SugaredLogger
}

// NewKubeLeaderElector New create client
func NewKubeLeaderElector(lockType, name, ns, kubeconfig string,
	leaseDuration, renewDuration, retryPeriod time.Duration,
) (LeaderElector, error) {
	var restConfig *rest.Config
	var err error

	cl := new(KubeLeaderElector)
	cl.lockType = lockType
	cl.name = name
	cl.namespace = ns
	cl.leaseDuration = leaseDuration
	cl.renewDuration = renewDuration
	cl.retryPeriod = retryPeriod
	cl.closeCh = make(chan struct{})
	cl.leadingCh = make(chan struct{})
	cl.logger = logging.GetLogger().Named("kube-leader-elector")

	id, err := os.Hostname()
	if err != nil {
		cl.logger.Error(err, "get hostname failed")
		return nil, err
	}

	// create kubernetes client for leader election
	if len(kubeconfig) != 0 {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			cl.logger.Error(err, fmt.Sprintf("create internal client with kubeconfig %s failed", kubeconfig))
			return nil, eris.Wrapf(err, "create internal client with kubeconfig %s failed", kubeconfig)
		}
	} else {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			cl.logger.Error(err, "build incluster config failed")
			return nil, eris.Wrapf(err, "buidl incluster config failed")
		}
	}
	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		cl.logger.Error(err, "create client set from config failed")
		return nil, eris.Wrapf(err, "create client set from config failed")
	}

	id = fmt.Sprintf("%s_%s_%s", id, uuid.NewUUID(), config.InstanceIP)

	rl, err := resourcelock.New(cl.lockType, cl.namespace, cl.name,
		k8sClientSet.CoreV1(), k8sClientSet.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		})
	if err != nil {
		cl.logger.Error(err, "create resource lock failed")
		return nil, err
	}
	cl.lock = rl

	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: cl.onStartedLeading,
			OnStoppedLeading: cl.onStoppedLeading,
		},
	})
	if err != nil {
		cl.logger.Error(err, "create client-go leader elector failed")
		return nil, eris.Wrapf(err, "create client-go leader elector failed")
	}
	cl.elector = elector
	return cl, nil
}

// Run run election
func (c *KubeLeaderElector) Run(ctx context.Context) {
	c.ctx = ctx
	go c.elector.Run(ctx)
}

func (c *KubeLeaderElector) onStartedLeading(ctx context.Context) {
	c.logger.Info("become leader")
	close(c.leadingCh)
}

func (c *KubeLeaderElector) onStoppedLeading() {
	c.logger.Info("become follower")
	close(c.closeCh)
	c.leadingCh = make(chan struct{})
	c.closeCh = make(chan struct{})
	go c.elector.Run(c.ctx)
}

// Leader ...
func (c *KubeLeaderElector) Leader() string {
	return c.elector.GetLeader()
}

// WaitForLeading ...
func (c *KubeLeaderElector) WaitForLeading() (closeCh <-chan struct{}) {
	if c.elector.IsLeader() {
		return c.closeCh
	}
	<-c.leadingCh
	return c.closeCh
}

// IsMaster to see if it is master
func (c *KubeLeaderElector) IsMaster() bool {
	return c.isMaster
}
