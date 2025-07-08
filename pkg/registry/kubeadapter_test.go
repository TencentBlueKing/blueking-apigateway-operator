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

package registry

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

var _ = Describe("Test KubeAdapter", Ordered, func() {
	var regAdapter *K8SRegistryAdapter
	var namespace string
	var builder *fake.ClientBuilder
	var scheme *runtime.Scheme

	BeforeEach(func() {
		namespace = "test"

		// Set up client
		builder = fake.NewClientBuilder()
		scheme = runtime.NewScheme()
		v1beta1.AddToScheme(scheme)
		builder.WithScheme(scheme)

		regAdapter = &K8SRegistryAdapter{
			kubeClient: builder.Build(),
			namespace:  namespace,
			logger:     logging.GetLogger(),
		}
	})

	It("Create normal", func() {
		reg, handler := NewK8SResourceRegistry(builder.Build(), "test")
		Expect(reg).ShouldNot(BeNil())
		Expect(handler).ShouldNot(BeNil())
	})

	It("Get normal", func() {
		// Init the client with a resource object, update the kubeClient field of adapter
		client := builder.WithObjects(&v1beta1.BkGatewayResource{
			ObjectMeta: v1.ObjectMeta{
				Namespace: namespace,
				Name:      "resource",
				Labels:    map[string]string{"test": "test"},
			},
		}).Build()
		regAdapter.kubeClient = client

		resultObj := v1beta1.BkGatewayResource{}
		err := regAdapter.Get(context.Background(), ResourceKey{ResourceName: "resource"}, &resultObj)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(resultObj.GetName()).Should(Equal("resource"))
		Expect(resultObj.GetLabels()).Should(Equal(map[string]string{"test": "test"}))
	})

	Context("Test List method", Ordered, func() {
		var client client.Client

		BeforeEach(func() {
			// Set the client with two initial resource objects
			client = builder.WithObjects(
				&v1beta1.BkGatewayResource{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "r-1",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "default",
							config.BKAPIGatewayLabelKeyGatewayStage: "stag",
							config.BKAPIGatewayLabelKeyResourceName: "r-1",
						},
					},
				},
				&v1beta1.BkGatewayResource{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "r-2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "default",
							config.BKAPIGatewayLabelKeyGatewayStage: "prod",
							config.BKAPIGatewayLabelKeyResourceName: "r-2",
						},
					},
				},
				&v1beta1.BkGatewayResource{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "r-3",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "another-gw",
							config.BKAPIGatewayLabelKeyGatewayStage: "prod",
							config.BKAPIGatewayLabelKeyResourceName: "r-3",
						},
					},
				},
			).Build()
			regAdapter.kubeClient = client
		})

		It("Empty selector", func() {
			objList := v1beta1.BkGatewayResourceList{}
			err := regAdapter.List(context.Background(), ResourceKey{}, &objList)

			Expect(err).Should(BeNil())
			Expect(len(objList.Items)).To(Equal(3))
		})

		It("With gateway name", func() {
			objList := &v1beta1.BkGatewayResourceList{}
			err := regAdapter.List(
				context.Background(),
				ResourceKey{StageInfo: StageInfo{GatewayName: "default"}},
				objList,
			)

			Expect(err).Should(BeNil())
			Expect(len(objList.Items)).To(Equal(2))
		})

		It("With gateway and stage name", func() {
			objList := &v1beta1.BkGatewayResourceList{}
			err := regAdapter.List(
				context.Background(),
				ResourceKey{StageInfo: StageInfo{GatewayName: "default", StageName: "prod"}},
				objList,
			)

			Expect(err).Should(BeNil())
			Expect(len(objList.Items)).To(Equal(1))
		})

		It("With resource name", func() {
			objList := &v1beta1.BkGatewayResourceList{}
			err := regAdapter.List(
				context.Background(),
				ResourceKey{
					StageInfo:    StageInfo{GatewayName: "default", StageName: "prod"},
					ResourceName: "r-2",
				},
				objList,
			)

			Expect(err).Should(BeNil())
			Expect(len(objList.Items)).To(Equal(1))
		})
	})

	Context("ListStages", Ordered, func() {
		var client client.Client

		BeforeEach(func() {
			// Set the client with two initial stage objects
			client = builder.WithObjects(
				&v1beta1.BkGatewayStage{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "stag",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "default",
							config.BKAPIGatewayLabelKeyGatewayStage: "stag",
						},
					},
				},
				&v1beta1.BkGatewayStage{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "prod",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "default",
							config.BKAPIGatewayLabelKeyGatewayStage: "prod",
						},
					},
				},
				// A stage object with no valid labels will be ignored.
				&v1beta1.BkGatewayStage{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "prod-invalid-labels",
						Labels:    map[string]string{},
					},
				},
			).Build()
			regAdapter.kubeClient = client
		})

		It("List stage normal", func() {
			objList, err := regAdapter.ListStages(context.Background())

			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(objList)).Should(Equal(2))
		})
	})

	Context("Watch", Ordered, func() {
		var watchCh <-chan *ResourceMetadata
		var cancel context.CancelFunc
		var ctx context.Context

		BeforeAll(func() {
			ctx, cancel = context.WithCancel(context.Background())
			watchCh = regAdapter.Watch(ctx)
		})

		It("will receive event", func() {
			regAdapter.KubeEventHandler(&ResourceMetadata{})

			rm := <-watchCh
			Expect(rm).NotTo(BeNil())
		})

		It("will close retch", func() {
			cancel()
			_, ok := <-watchCh
			Expect(ok).To(BeFalse())
		})

		It("will not block when there is no watcher", func() {
			doneCtx, doneCtxCancel := context.WithCancel(context.Background())
			go func() {
				regAdapter.KubeEventHandler(&ResourceMetadata{})
				doneCtxCancel()
			}()

			select {
			case <-doneCtx.Done():
				Expect(true).To(BeTrue())
			case <-time.After(time.Second):
				Expect("function should not block").To(BeEmpty())
			}
		})
	})
})
