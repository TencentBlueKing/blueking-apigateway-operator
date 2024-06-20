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

package leaderelection_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/tests/util"
)

var _ = Describe("EtcdLeaderElector", func() {
	var (
		elector1 leaderelection.LeaderElector
		elector2 leaderelection.LeaderElector

		etcdClient *clientv3.Client
		err        error
		ctx        = context.Background()
	)

	// 使用 etcd mock 客户端
	etcdClient, _, err = util.StartEmbedEtcdClient(ctx)
	Expect(err).To(BeNil())

	BeforeEach(func() {
		config.InstanceName = "test-instance1"
		elector1, err = leaderelection.NewEtcdLeaderElector(etcdClient, "test-prefix")
		Expect(err).To(BeNil())
		config.InstanceName = "test-instance2"
		elector2, err = leaderelection.NewEtcdLeaderElector(etcdClient, "test-prefix")
		Expect(err).To(BeNil())

	})

	Describe("NewEtcdLeaderElector", func() {
		It("should create a new EtcdLeaderElector without error", func() {
			Expect(elector1).NotTo(BeNil())
			Expect(elector2).NotTo(BeNil())
		})

	})

	Describe("Run", func() {
		It("should run the election process", func() {
			go elector1.Run(context.Background())
			go elector2.Run(context.Background())

			Expect(elector1.Leader()).To(Equal(elector2.Leader()))
		})
	})
})

var _ = Describe("KubeLeaderElector", func() {
	var (
		elector1 leaderelection.LeaderElector
		elector2 leaderelection.LeaderElector

		err error
	)

	metric.InitMetric(prometheus.DefaultRegisterer)
	// mock kube client
	fakeKubeClient := fake.NewSimpleClientset()

	BeforeEach(func() {
		elector1, err = leaderelection.NewKubeLeaderElector(
			"leases",
			"election.gateway.bk.tencent.com",
			"test-namespace",
			fakeKubeClient,
			30,
			25, 5,
		)
		Expect(err).To(BeNil())
		elector2, err = leaderelection.NewKubeLeaderElector(
			"leases",
			"election.gateway.bk.tencent.com",
			"test-namespace",
			fakeKubeClient,
			30,
			25, 5,
		)
		Expect(err).To(BeNil())

	})

	Describe("Run", func() {
		It("should run the election process", func() {
			go elector1.Run(context.Background())
			go elector2.Run(context.Background())
			<-elector1.WaitForLeading()
			<-elector2.WaitForLeading()
			Expect(elector1.Leader()).To(Equal(elector2.Leader()))
		})
	})
})
