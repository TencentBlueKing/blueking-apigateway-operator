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

package conversion

import (
	"testing"
	"time"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

func Test_calculateMatchSubPathRoutePriority(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test",
			args: args{
				path: "a/b/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 5,
		},
		{
			name: "test_with_args",
			args: args{
				path: "a/:abc/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 5,
		},
		{
			name: "test_empty",
			args: args{
				path: "",
			},
			want: MATCH_SUB_PATH_PRIORITY,
		},
		{
			name: "test_ok",
			args: args{
				path: "a/abc/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateMatchSubPathRoutePriority(tt.args.path); got != tt.want {
				t.Errorf("calculateMatchSubPathRoutePriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

var _ = Describe("resource", func() {
	Describe("convertHTTPTimeout", func() {
		It("nil", func() {
			c := &Converter{}
			timeout := c.convertHTTPTimeout(nil)
			gomega.Expect(timeout).To(gomega.BeNil())
		})

		It("less then zero", func() {
			c := &Converter{}
			timeout := c.convertHTTPTimeout(&v1beta1.UpstreamTimeout{
				Connect: v1beta1.FormatDuration(time.Second * -1),
				Send:    v1beta1.FormatDuration(time.Second * -1),
				Read:    v1beta1.FormatDuration(time.Second * -1),
			})
			gomega.Expect(timeout).To(gomega.Equal(&apisixv1.UpstreamTimeout{
				Connect: 60,
				Read:    60,
				Send:    60,
			}))
		})

		It("ok", func() {
			c := &Converter{}
			timeout := c.convertHTTPTimeout(&v1beta1.UpstreamTimeout{
				Connect: v1beta1.FormatDuration(time.Second * 1),
				Send:    v1beta1.FormatDuration(time.Second * 1),
				Read:    v1beta1.FormatDuration(time.Second * 1),
			})
			gomega.Expect(timeout).To(gomega.Equal(&apisixv1.UpstreamTimeout{
				Connect: 1,
				Read:    1,
				Send:    1,
			}))
		})
	})

	Describe("appendStagePlugins", func() {
		It("nil", func() {
			c := &Converter{
				stage: &v1beta1.BkGatewayStage{
					Spec: v1beta1.BkGatewayStageSpec{
						Plugins: nil,
					},
				},
			}

			stagePlugins := make(map[string]interface{})
			c.appendStagePlugins(stagePlugins)
			gomega.Expect(stagePlugins).To(gomega.HaveLen(0))
		})

		It("ok", func() {
			c := &Converter{
				stage: &v1beta1.BkGatewayStage{
					Spec: v1beta1.BkGatewayStageSpec{
						Plugins: []*v1beta1.BkGatewayPlugin{
							nil,
							{
								Name: "plugin",
								Config: runtime.RawExtension{
									Raw: []byte(`{"headers": {}}`),
								},
							},
						},
					},
				},
			}

			stagePlugins := make(map[string]interface{})
			c.appendStagePlugins(stagePlugins)
			gomega.Expect(stagePlugins).To(gomega.HaveLen(1))
		})
	})

	Describe("getProxyRewrite", func() {
		It("rewrite not enable", func() {
			c := &Converter{}
			rewrite, err := c.getProxyRewrite(
				&v1beta1.BkGatewayResourceHTTPRewrite{Enabled: false},
				&v1beta1.BkGatewayResource{},
			)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rewrite).To(gomega.HaveLen(0))
		})

		It("rewrite with path", func() {
			c := &Converter{
				stage: &v1beta1.BkGatewayStage{
					Spec: v1beta1.BkGatewayStageSpec{
						Vars: map[string]string{},
						Rewrite: &v1beta1.BkGatewayRewrite{
							Enabled: true,
							Headers: map[string]string{
								"test": "test",
							},
						},
					},
				},
			}
			rewrite, err := c.getProxyRewrite(
				&v1beta1.BkGatewayResourceHTTPRewrite{
					Enabled: true,
					Path:    "/rewrite",
					Method:  "GET",
				},
				&v1beta1.BkGatewayResource{
					Spec: v1beta1.BkGatewayResourceSpec{
						MatchSubPath: true,
					},
				},
			)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rewrite).To(gomega.Equal(map[string]interface{}{
				"uri":                "/rewrite/${bk_api_subpath_match_param_name}",
				"match_subpath":      true,
				"subpath_param_name": "bk_api_subpath_match_param_name",
				"method":             "GET",
			}))
		})

		It("rewrite without path", func() {
			c := &Converter{
				stage: &v1beta1.BkGatewayStage{
					Spec: v1beta1.BkGatewayStageSpec{
						Vars: map[string]string{},
						Rewrite: &v1beta1.BkGatewayRewrite{
							Enabled: true,
							Headers: map[string]string{
								"test": "test",
							},
						},
					},
				},
			}
			rewrite, err := c.getProxyRewrite(
				&v1beta1.BkGatewayResourceHTTPRewrite{
					Enabled: true,
					Method:  "GET",
				},
				&v1beta1.BkGatewayResource{
					Spec: v1beta1.BkGatewayResourceSpec{
						MatchSubPath: true,
					},
				},
			)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rewrite).To(gomega.Equal(map[string]interface{}{
				"method": "GET",
			}))
		})

		It("rewrite without path match", func() {
			c := &Converter{
				stage: &v1beta1.BkGatewayStage{
					Spec: v1beta1.BkGatewayStageSpec{
						Vars: map[string]string{},
						Rewrite: &v1beta1.BkGatewayRewrite{
							Enabled: true,
						},
					},
				},
			}
			rewrite, err := c.getProxyRewrite(
				&v1beta1.BkGatewayResourceHTTPRewrite{
					Enabled: true,
					Path:    "/rewrite",
					Method:  "GET",
				},
				&v1beta1.BkGatewayResource{
					Spec: v1beta1.BkGatewayResourceSpec{
						MatchSubPath: false,
					},
				},
			)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rewrite).To(gomega.Equal(map[string]interface{}{
				"uri":    "/rewrite",
				"method": "GET",
			}))
		})
	})

	It("rewrite not enable", func() {
		c := &Converter{
			stage: &v1beta1.BkGatewayStage{
				Spec: v1beta1.BkGatewayStageSpec{
					Vars: map[string]string{},
				},
			},
			logger: logging.GetLogger().Named("converter"),
		}
		upstream, err := c.convertUpstream(
			metav1.TypeMeta{},
			metav1.ObjectMeta{},
			&v1beta1.BkGatewayUpstreamConfig{
				Checks:       &v1beta1.UpstreamHealthCheck{},
				Type:         "roundrobin",
				HashOn:       "none",
				Key:          "none",
				Scheme:       "https",
				PassHost:     "pass",
				UpstreamHost: "127.0.0.1",
				RetryTimeout: utils.IntPtr(5),
				Timeout: &v1beta1.UpstreamTimeout{
					Connect: v1beta1.FormatDuration(time.Second * 1),
					Send:    v1beta1.FormatDuration(time.Second * 1),
					Read:    v1beta1.FormatDuration(time.Second * 1),
				},
				Nodes: []v1beta1.BkGatewayNode{
					{
						Host:     "127.0.0.1",
						Port:     80,
						Weight:   1,
						Priority: utils.IntPtr(0),
					},
				},
				TLSEnable: false,
			},
		)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(upstream).To(gomega.Equal(&apisix.Upstream{
			Type:         utils.StringPtr("roundrobin"),
			HashOn:       utils.StringPtr("none"),
			Key:          utils.StringPtr("none"),
			Nodes:        []v1beta1.BkGatewayNode{{Host: "127.0.0.1", Port: 80, Weight: 1, Priority: utils.IntPtr(0)}},
			Scheme:       utils.StringPtr("https"),
			RetryTimeout: utils.IntPtr(5),
			PassHost:     utils.StringPtr("pass"),
			UpstreamHost: utils.StringPtr("127.0.0.1"),
			Timeout:      &apisixv1.UpstreamTimeout{Connect: 1, Send: 1, Read: 1},
		}))
	})

	It("convertUpstream", func() {
		c := &Converter{
			stage: &v1beta1.BkGatewayStage{
				Spec: v1beta1.BkGatewayStageSpec{
					Vars: map[string]string{},
				},
			},
			logger: logging.GetLogger().Named("converter"),
		}
		upstream, err := c.convertUpstream(
			metav1.TypeMeta{},
			metav1.ObjectMeta{},
			&v1beta1.BkGatewayUpstreamConfig{
				Checks:       &v1beta1.UpstreamHealthCheck{},
				Type:         "roundrobin",
				HashOn:       "none",
				Key:          "none",
				Scheme:       "https",
				PassHost:     "pass",
				UpstreamHost: "127.0.0.1",
				RetryTimeout: utils.IntPtr(5),
				Timeout: &v1beta1.UpstreamTimeout{
					Connect: v1beta1.FormatDuration(time.Second * 1),
					Send:    v1beta1.FormatDuration(time.Second * 1),
					Read:    v1beta1.FormatDuration(time.Second * 1),
				},
				Nodes: []v1beta1.BkGatewayNode{
					{
						Host:     "127.0.0.1",
						Port:     80,
						Weight:   1,
						Priority: utils.IntPtr(0),
					},
				},
				TLSEnable: false,
			},
		)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(upstream).To(gomega.Equal(&apisix.Upstream{
			Type:         utils.StringPtr("roundrobin"),
			HashOn:       utils.StringPtr("none"),
			Key:          utils.StringPtr("none"),
			Nodes:        []v1beta1.BkGatewayNode{{Host: "127.0.0.1", Port: 80, Weight: 1, Priority: utils.IntPtr(0)}},
			Scheme:       utils.StringPtr("https"),
			RetryTimeout: utils.IntPtr(5),
			PassHost:     utils.StringPtr("pass"),
			UpstreamHost: utils.StringPtr("127.0.0.1"),
			Timeout:      &apisixv1.UpstreamTimeout{Connect: 1, Send: 1, Read: 1},
		}))
	})

	It("convertPlugin", func() {
		c := &Converter{
			logger: logging.GetLogger().Named("converter"),
		}
		name, config := c.convertPlugin(
			&v1beta1.BkGatewayPlugin{
				Name: "test",
				Config: runtime.RawExtension{
					Raw: []byte(`{"test":"test"}`),
				},
			},
		)
		gomega.Expect(name).To(gomega.Equal("test"))
		gomega.Expect(config).To(gomega.Equal(map[string]interface{}{"test": "test"}))
	})

	It("convertResource", func() {
		c := &Converter{
			stage: &v1beta1.BkGatewayStage{
				Spec: v1beta1.BkGatewayStageSpec{
					Vars: map[string]string{},
				},
			},
			logger: logging.GetLogger().Named("converter"),
		}
		route, err := c.convertResource(
			&v1beta1.BkGatewayResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "test",
				},
				Spec: v1beta1.BkGatewayResourceSpec{
					Desc:    "test resource",
					Methods: []string{"GET"},
					Timeout: &v1beta1.UpstreamTimeout{
						Connect: v1beta1.FormatDuration(time.Second),
						Send:    v1beta1.FormatDuration(time.Second),
						Read:    v1beta1.FormatDuration(time.Second),
					},
					URI:          "/test-resource",
					MatchSubPath: false,
					Upstream: &v1beta1.BkGatewayUpstreamConfig{
						Type: "roundrobin",
						Nodes: []v1beta1.BkGatewayNode{
							{
								Host:     "127.0.0.1:9090",
								Port:     8080,
								Weight:   10,
								Priority: utils.IntPtr(-1),
							},
						},
					},
				},
			},
			[]*v1beta1.BkGatewayService{},
		)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(route).To(gomega.Equal(&apisix.Route{
			Route: apisixv1.Route{
				Metadata: apisixv1.Metadata{
					ID:   "..test-resource",
					Name: "test-resource",
					Desc: "test resource",
					Labels: map[string]string{
						config.BKAPIGatewayLabelKeyGatewayName:  "",
						config.BKAPIGatewayLabelKeyGatewayStage: "",
					},
				},
				Host:    "",
				Uris:    []string{"/test-resource", "/test-resource/"},
				Methods: []string{"GET"},
				Timeout: &apisixv1.UpstreamTimeout{
					Connect: 1,
					Read:    1,
					Send:    1,
				},
			},
			Status: utils.IntPtr(1),
			Upstream: &apisix.Upstream{
				Type:  utils.StringPtr("roundrobin"),
				Nodes: []v1beta1.BkGatewayNode{{Host: "127.0.0.1", Port: 9090, Weight: 10, Priority: utils.IntPtr(-1)}},
			},
		}))
	})
})
