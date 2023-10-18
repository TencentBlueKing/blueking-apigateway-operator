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
	"context"
	"testing"
	"time"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

func marshallIgnoreErr(obj interface{}) string {
	by, _ := json.Marshal(obj)
	return string(by)
}

func getStage() *v1beta1.BkGatewayStage {
	return &v1beta1.BkGatewayStage{
		ObjectMeta: metav1.ObjectMeta{
			Name: "stage",
		},
		Spec: v1beta1.BkGatewayStageSpec{
			Name:       "stage",
			Domain:     "test.exmaple.com",
			PathPrefix: "/",
			Desc:       "test desc",
			Vars: map[string]string{
				"runMode": "prod",
			},
			Rewrite: &v1beta1.BkGatewayRewrite{
				Enabled: true,
				Headers: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			Plugins: []*v1beta1.BkGatewayPlugin{
				{
					Name: "limit-req",
					Config: runtime.RawExtension{
						Raw: []byte("{\"rate\":1,\"burst\":2,\"rejected_code\":429,\"key\":\"consumer_name\"}"),
					},
				},
			},
		},
	}
}

// a.com, *.b.com
var cert1 = v1beta1.TLSCert{
	Cert: `-----BEGIN CERTIFICATE-----
MIIC0zCCAnqgAwIBAgIRAMBd1QC/ehbpIbYG7npNmMowCgYIKoZIzj0EAwIwHjEc
MBoGA1UEAxMTYmNzLWNhLWJjcy1zZXJ2aWNlczAeFw0yMjA1MTEwODE5MzVaFw0z
MDA3MjgwODE5MzVaMHExCzAJBgNVBAYTAkNOMQswCQYDVQQIEwJHRDELMAkGA1UE
BxMCU1oxDTALBgNVBAkTBFlISkQxEDAOBgNVBAoTB1RlbmNlbnQxETAPBgNVBAsT
CEJsdWVLaW5nMRQwEgYDVQQDEwtleGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAMS61bhXZ01TPTcS/XvcUQFsv1PEE/4bkW1wL6Vckrtp
3U5grHBiW8cSJwJZ9uGvBiB9GY1s7+elbhVJo2TbzZCqncGGX3OYj1MQCfIEznW+
cyQmWDKv1TCM3Ka45sEs/aI8Rpf/Hm4kVntCZ6J051QKhNXNQGMYkoonnzKKSnQn
Po4OAvqzIY8E/g0p+WPZBN+a7Myx8y/u89unC02Qq5Zc5ro8STs9ZiiY5GupKC8d
XhfKncntQZ7omQfFOtAqaBuKm5RoRgGV3W1n2Du020PL36km3UkeHcQc0Q7EXk6s
GQRFDylhArB6wDGXX/lMcxzIvT+Iqy9Qny7o18nNn/sCAwEAAaN7MHkwDgYDVR0P
AQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMB
Af8EAjAAMB8GA1UdIwQYMBaAFALXIo07GzdEZACFQyBjz4rV6JhJMBkGA1UdEQQS
MBCCBWEuY29tggcqLmIuY29tMAoGCCqGSM49BAMCA0cAMEQCICPV45uE3yvMEm6C
4eargjhi7vxnHLtsSeSUSXEvOkijAiBAG2qZ/tpwahcjUjHdxAaNvT+yxvkfz+M1
OMbRkgdACQ==
-----END CERTIFICATE-----`,
	Key: `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDEutW4V2dNUz03
Ev173FEBbL9TxBP+G5FtcC+lXJK7ad1OYKxwYlvHEicCWfbhrwYgfRmNbO/npW4V
SaNk282Qqp3Bhl9zmI9TEAnyBM51vnMkJlgyr9UwjNymuObBLP2iPEaX/x5uJFZ7
QmeidOdUCoTVzUBjGJKKJ58yikp0Jz6ODgL6syGPBP4NKflj2QTfmuzMsfMv7vPb
pwtNkKuWXOa6PEk7PWYomORrqSgvHV4Xyp3J7UGe6JkHxTrQKmgbipuUaEYBld1t
Z9g7tNtDy9+pJt1JHh3EHNEOxF5OrBkERQ8pYQKwesAxl1/5THMcyL0/iKsvUJ8u
6NfJzZ/7AgMBAAECggEBAKIiSRlQD3cO7xiAsiBuhuRht51VsBRwq/5Bw0LJdLS4
nweFbRiCN5ltQHETrAB7utTzxSdlbKLBGGS698qbzGM5+iIQmIIwbY7LXSb1ByLK
/yH/6Bh+CXml3gQZxzPV3ILkolmKjI3BrPSQ2dBuAGim8qsyKaqCCeOKnA2PI6Vt
cB30nO0BO0WziccZPBIR/vzTcHuh+em2cfz1kV+6Yt/n7R5Ds2JmC4RfTVTv7XyC
/FNAoLVbzLfoJmcAjCFghSQSezQDlR+M0eDdEHbBPAosSUsfqFH6/jtjdlFVF230
Q+u364GoD/NR6Em8tv3CAryQONrUKjQbHWxKhgKOHcECgYEA7Ep73QxlEzUpOIsy
k2aytjUxVU57sQeTA6igpRIgxvab8DL17WTXz/iwfDekqPzisfHJ9v51ggwv8PhS
pvgzDbsRP/emz0Z1UlVNukrcW6ZsoQGvCrQPoVGdsjwtsERff/GS9d4STfTQQSWL
Oi0tvdD7kalz+gtYSxY5GjNT18kCgYEA1SOaXNPnczKZvZu8a/H5H6J1DjxusPkX
4LZ8sN3bmrZDc4NYs/d99WLzTN0RF8sI03B3zrZm9cI1eE6pOIKcYIRFHGk2bxNj
zV2vhkpmFpEuFlTqyoRbcE1nvHX6V6ZUx3vvvMSbicaFQ13ZhjqRoj/TD0kL7ZlC
9jwGnhx046MCgYEAparPTztKfn4OSZumuRwO/psq3JGrPYJ++9i10SZ1nqn2ySEh
tfC3MxQ8wMrOgsDTPEm2/ZqIzsY2sq+YW4K3YNAglwXOiZLv3Or8FTo5Z3S2wugI
TuvR7ZvogbeZnPVDM9Qu4n1xvgCAJrzo8cANSwGD8Curqctce0C4hnsoNKkCgYEA
r3S1mAEhISXghcP0YnA5gp87+VIqVSlZTLUtBHQ+Waf88tSHau8sE5s3amj5rzqG
s3h8SADD1T/gwH8QsuJiVNnOAsth8iJmICMlYUlRrPYqmFujRL+cfmBaKzx7rzfP
xr/x5NV8rPhtr71MWkFQrd4Yoxag6SEnjIhxcis+1j0CgYEAziD8DUhIler3rWKD
LauBB0ZLBAyzA0W0t8Eb01y7+ROGB04k7kDWkKkkJGGT8JeyA9Be0FQ8bKsnKhbo
6dKMJq8xwBiDS3vrJyNEzPHPnThs5zDJdeBZrn6qjnSERfDYRnm9CDJuOP5+VsdQ
c9TCEaBGVFyH5ZMhzcSvLq8k8bU=
-----END PRIVATE KEY-----`,
}

// a.a.com, *.b.com
var cert2 = v1beta1.TLSCert{
	Cert: `-----BEGIN CERTIFICATE-----
MIIC1TCCAnugAwIBAgIQNz/VdMR2eGXtflpl2muEdzAKBggqhkjOPQQDAjAeMRww
GgYDVQQDExNiY3MtY2EtYmNzLXNlcnZpY2VzMB4XDTIyMDUxMTA4MjEwM1oXDTMw
MDcyODA4MjEwM1owcTELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkdEMQswCQYDVQQH
EwJTWjENMAsGA1UECRMEWUhKRDEQMA4GA1UEChMHVGVuY2VudDERMA8GA1UECxMI
Qmx1ZUtpbmcxFDASBgNVBAMTC2V4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAxLrVuFdnTVM9NxL9e9xRAWy/U8QT/huRbXAvpVySu2nd
TmCscGJbxxInAln24a8GIH0ZjWzv56VuFUmjZNvNkKqdwYZfc5iPUxAJ8gTOdb5z
JCZYMq/VMIzcprjmwSz9ojxGl/8ebiRWe0JnonTnVAqE1c1AYxiSiiefMopKdCc+
jg4C+rMhjwT+DSn5Y9kE35rszLHzL+7z26cLTZCrllzmujxJOz1mKJjka6koLx1e
F8qdye1BnuiZB8U60CpoG4qblGhGAZXdbWfYO7TbQ8vfqSbdSR4dxBzRDsReTqwZ
BEUPKWECsHrAMZdf+UxzHMi9P4irL1CfLujXyc2f+wIDAQABo30wezAOBgNVHQ8B
Af8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMAwGA1UdEwEB
/wQCMAAwHwYDVR0jBBgwFoAUAtcijTsbN0RkAIVDIGPPitXomEkwGwYDVR0RBBQw
EoIHYS5hLmNvbYIHKi5iLmNvbTAKBggqhkjOPQQDAgNIADBFAiA5t8qB9lSBpdA1
f8xjagaJjK1ec0P5euultTq1XLt4BwIhAJndkGiTFDshH9WntNCLltLUQDPaDqM0
gNClQrlM7gLP
-----END CERTIFICATE-----`,
	Key: `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDEutW4V2dNUz03
Ev173FEBbL9TxBP+G5FtcC+lXJK7ad1OYKxwYlvHEicCWfbhrwYgfRmNbO/npW4V
SaNk282Qqp3Bhl9zmI9TEAnyBM51vnMkJlgyr9UwjNymuObBLP2iPEaX/x5uJFZ7
QmeidOdUCoTVzUBjGJKKJ58yikp0Jz6ODgL6syGPBP4NKflj2QTfmuzMsfMv7vPb
pwtNkKuWXOa6PEk7PWYomORrqSgvHV4Xyp3J7UGe6JkHxTrQKmgbipuUaEYBld1t
Z9g7tNtDy9+pJt1JHh3EHNEOxF5OrBkERQ8pYQKwesAxl1/5THMcyL0/iKsvUJ8u
6NfJzZ/7AgMBAAECggEBAKIiSRlQD3cO7xiAsiBuhuRht51VsBRwq/5Bw0LJdLS4
nweFbRiCN5ltQHETrAB7utTzxSdlbKLBGGS698qbzGM5+iIQmIIwbY7LXSb1ByLK
/yH/6Bh+CXml3gQZxzPV3ILkolmKjI3BrPSQ2dBuAGim8qsyKaqCCeOKnA2PI6Vt
cB30nO0BO0WziccZPBIR/vzTcHuh+em2cfz1kV+6Yt/n7R5Ds2JmC4RfTVTv7XyC
/FNAoLVbzLfoJmcAjCFghSQSezQDlR+M0eDdEHbBPAosSUsfqFH6/jtjdlFVF230
Q+u364GoD/NR6Em8tv3CAryQONrUKjQbHWxKhgKOHcECgYEA7Ep73QxlEzUpOIsy
k2aytjUxVU57sQeTA6igpRIgxvab8DL17WTXz/iwfDekqPzisfHJ9v51ggwv8PhS
pvgzDbsRP/emz0Z1UlVNukrcW6ZsoQGvCrQPoVGdsjwtsERff/GS9d4STfTQQSWL
Oi0tvdD7kalz+gtYSxY5GjNT18kCgYEA1SOaXNPnczKZvZu8a/H5H6J1DjxusPkX
4LZ8sN3bmrZDc4NYs/d99WLzTN0RF8sI03B3zrZm9cI1eE6pOIKcYIRFHGk2bxNj
zV2vhkpmFpEuFlTqyoRbcE1nvHX6V6ZUx3vvvMSbicaFQ13ZhjqRoj/TD0kL7ZlC
9jwGnhx046MCgYEAparPTztKfn4OSZumuRwO/psq3JGrPYJ++9i10SZ1nqn2ySEh
tfC3MxQ8wMrOgsDTPEm2/ZqIzsY2sq+YW4K3YNAglwXOiZLv3Or8FTo5Z3S2wugI
TuvR7ZvogbeZnPVDM9Qu4n1xvgCAJrzo8cANSwGD8Curqctce0C4hnsoNKkCgYEA
r3S1mAEhISXghcP0YnA5gp87+VIqVSlZTLUtBHQ+Waf88tSHau8sE5s3amj5rzqG
s3h8SADD1T/gwH8QsuJiVNnOAsth8iJmICMlYUlRrPYqmFujRL+cfmBaKzx7rzfP
xr/x5NV8rPhtr71MWkFQrd4Yoxag6SEnjIhxcis+1j0CgYEAziD8DUhIler3rWKD
LauBB0ZLBAyzA0W0t8Eb01y7+ROGB04k7kDWkKkkJGGT8JeyA9Be0FQ8bKsnKhbo
6dKMJq8xwBiDS3vrJyNEzPHPnThs5zDJdeBZrn6qjnSERfDYRnm9CDJuOP5+VsdQ
c9TCEaBGVFyH5ZMhzcSvLq8k8bU=
-----END PRIVATE KEY-----`,
}

var obj1 = registry.ResourceKey{
	StageInfo: registry.StageInfo{
		GatewayName: "testgateway",
		StageName:   "testns",
	},
	ResourceName: "obj1",
}

var obj2 = registry.ResourceKey{
	StageInfo: registry.StageInfo{
		GatewayName: "testgateway",
		StageName:   "testns",
	},
	ResourceName: "obj2",
}

func getRadixTree(t *testing.T) radixtree.RadixTree {
	tree := radixtree.NewSuffixRadixTree()
	var nilErr error
	// a.com, *.b.com
	ok, err := tree.Insert(obj1, &cert1)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)
	ok, err = tree.Insert(obj2, &cert2)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)
	return tree
}

func TestConvert(t *testing.T) {
	testCases := []struct {
		title          string
		inputStage     *v1beta1.BkGatewayStage
		inputResources []*v1beta1.BkGatewayResource
		inputServices  []*v1beta1.BkGatewayService
		hasErr         bool
		outConfig      *apisix.ApisixConfiguration
	}{
		{
			title:      "convert empty stage",
			inputStage: getStage(),
			outConfig:  apisix.NewEmptyApisixConfiguration(),
		},
		{
			title:      "convert stage resource",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
			},
			outConfig: &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"gateway.stage.test-resource": {
						Route: apisixv1.Route{
							Metadata: apisixv1.Metadata{
								ID:   "gateway.stage.test-resource",
								Name: "test-resource",
								Desc: "test resource",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "stage",
								},
							},
							Host:    "test.exmaple.com",
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
							Type: utils.StringPtr("roundrobin"),
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1",
									Port:     9090,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
						},
					},
				},
				Services:        make(map[string]*apisix.Service),
				PluginMetadatas: make(map[string]*apisix.PluginMetadata),
				SSLs:            make(map[string]*apisix.SSL),
			},
		},
		{
			title:      "convert path parameter",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
						URI:          "/test-resource/{env.runMode}/{var}",
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
			},
			outConfig: &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"gateway.stage.test-resource": {
						Route: apisixv1.Route{
							Metadata: apisixv1.Metadata{
								ID:   "gateway.stage.test-resource",
								Name: "test-resource",
								Desc: "test resource",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "stage",
								},
							},
							Host:    "test.exmaple.com",
							Uri:     "/test-resource/prod/:var/?",
							Methods: []string{"GET"},
							Timeout: &apisixv1.UpstreamTimeout{
								Connect: 1,
								Read:    1,
								Send:    1,
							},
						},
						Status: utils.IntPtr(1),
						Upstream: &apisix.Upstream{
							Type: utils.StringPtr("roundrobin"),
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1",
									Port:     9090,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
						},
					},
				},
				Services:        make(map[string]*apisix.Service),
				PluginMetadatas: make(map[string]*apisix.PluginMetadata),
				SSLs:            make(map[string]*apisix.SSL),
			},
		},
		{
			title:      "convert match subpath",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
						URI:          "/test-resource/{env.runMode}/{var}",
						MatchSubPath: true,
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
			},
			outConfig: &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"gateway.stage.test-resource": {
						Route: apisixv1.Route{
							Metadata: apisixv1.Metadata{
								ID:   "gateway.stage.test-resource",
								Name: "test-resource",
								Desc: "test resource",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "stage",
								},
							},
							Host: "test.exmaple.com",
							Uris: []string{
								"/test-resource/prod/:var",
								"/test-resource/prod/:var/*" + config.BKAPIGatewaySubpathMatchParamName,
							},
							Methods: []string{"GET"},
							Timeout: &apisixv1.UpstreamTimeout{
								Connect: 1,
								Read:    1,
								Send:    1,
							},
							Priority: -979,
						},
						Status: utils.IntPtr(1),
						Upstream: &apisix.Upstream{
							Type: utils.StringPtr("roundrobin"),
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1",
									Port:     9090,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
						},
					},
				},
				Services:        make(map[string]*apisix.Service),
				PluginMetadatas: make(map[string]*apisix.PluginMetadata),
				SSLs:            make(map[string]*apisix.SSL),
			},
		},
	}

	for index, test := range testCases {
		t.Logf("test %d: %s", index, test.title)

		tmpConverter, err := NewConverter("", "gateway", test.inputStage, nil, nil)
		if err != nil {
			t.Fatalf("expect no err but get err %s", err.Error())
			continue
		}
		apisixConf, err := tmpConverter.Convert(context.Background(), test.inputResources, test.inputServices, nil, nil)
		if err != nil {
			if test.hasErr == true {
				continue
			}
			t.Fatalf("expect no err but get err %s", err.Error())
		}
		assert.Equal(t, test.outConfig, apisixConf)
	}
}

func TestConvertForCert(t *testing.T) {
	testCases := []struct {
		title          string
		inputStage     *v1beta1.BkGatewayStage
		inputResources []*v1beta1.BkGatewayResource
		inputServices  []*v1beta1.BkGatewayService
		hasErr         bool
		outConfig      *apisix.ApisixConfiguration
	}{
		{
			title:      "convert cert with host succ",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
									Host:     "127.0.0.1",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost:     "rewrite",
							UpstreamHost: "a.com",
							TLSEnable:    true,
						},
					},
				},
			},
			outConfig: &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"gateway.stage.test-resource": {
						Route: apisixv1.Route{
							Metadata: apisixv1.Metadata{
								ID:   "gateway.stage.test-resource",
								Name: "test-resource",
								Desc: "test resource",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "stage",
								},
							},
							Host:    "test.exmaple.com",
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
							Type: utils.StringPtr("roundrobin"),
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost:     func() *string { ret := "rewrite"; return &ret }(),
							UpstreamHost: func() *string { ret := "a.com"; return &ret }(),
							TLS: &apisix.UpstreamTLS{
								ClientCert: `-----BEGIN CERTIFICATE-----
MIIC0zCCAnqgAwIBAgIRAMBd1QC/ehbpIbYG7npNmMowCgYIKoZIzj0EAwIwHjEc
MBoGA1UEAxMTYmNzLWNhLWJjcy1zZXJ2aWNlczAeFw0yMjA1MTEwODE5MzVaFw0z
MDA3MjgwODE5MzVaMHExCzAJBgNVBAYTAkNOMQswCQYDVQQIEwJHRDELMAkGA1UE
BxMCU1oxDTALBgNVBAkTBFlISkQxEDAOBgNVBAoTB1RlbmNlbnQxETAPBgNVBAsT
CEJsdWVLaW5nMRQwEgYDVQQDEwtleGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAMS61bhXZ01TPTcS/XvcUQFsv1PEE/4bkW1wL6Vckrtp
3U5grHBiW8cSJwJZ9uGvBiB9GY1s7+elbhVJo2TbzZCqncGGX3OYj1MQCfIEznW+
cyQmWDKv1TCM3Ka45sEs/aI8Rpf/Hm4kVntCZ6J051QKhNXNQGMYkoonnzKKSnQn
Po4OAvqzIY8E/g0p+WPZBN+a7Myx8y/u89unC02Qq5Zc5ro8STs9ZiiY5GupKC8d
XhfKncntQZ7omQfFOtAqaBuKm5RoRgGV3W1n2Du020PL36km3UkeHcQc0Q7EXk6s
GQRFDylhArB6wDGXX/lMcxzIvT+Iqy9Qny7o18nNn/sCAwEAAaN7MHkwDgYDVR0P
AQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMB
Af8EAjAAMB8GA1UdIwQYMBaAFALXIo07GzdEZACFQyBjz4rV6JhJMBkGA1UdEQQS
MBCCBWEuY29tggcqLmIuY29tMAoGCCqGSM49BAMCA0cAMEQCICPV45uE3yvMEm6C
4eargjhi7vxnHLtsSeSUSXEvOkijAiBAG2qZ/tpwahcjUjHdxAaNvT+yxvkfz+M1
OMbRkgdACQ==
-----END CERTIFICATE-----`,
								ClientKey: `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDEutW4V2dNUz03
Ev173FEBbL9TxBP+G5FtcC+lXJK7ad1OYKxwYlvHEicCWfbhrwYgfRmNbO/npW4V
SaNk282Qqp3Bhl9zmI9TEAnyBM51vnMkJlgyr9UwjNymuObBLP2iPEaX/x5uJFZ7
QmeidOdUCoTVzUBjGJKKJ58yikp0Jz6ODgL6syGPBP4NKflj2QTfmuzMsfMv7vPb
pwtNkKuWXOa6PEk7PWYomORrqSgvHV4Xyp3J7UGe6JkHxTrQKmgbipuUaEYBld1t
Z9g7tNtDy9+pJt1JHh3EHNEOxF5OrBkERQ8pYQKwesAxl1/5THMcyL0/iKsvUJ8u
6NfJzZ/7AgMBAAECggEBAKIiSRlQD3cO7xiAsiBuhuRht51VsBRwq/5Bw0LJdLS4
nweFbRiCN5ltQHETrAB7utTzxSdlbKLBGGS698qbzGM5+iIQmIIwbY7LXSb1ByLK
/yH/6Bh+CXml3gQZxzPV3ILkolmKjI3BrPSQ2dBuAGim8qsyKaqCCeOKnA2PI6Vt
cB30nO0BO0WziccZPBIR/vzTcHuh+em2cfz1kV+6Yt/n7R5Ds2JmC4RfTVTv7XyC
/FNAoLVbzLfoJmcAjCFghSQSezQDlR+M0eDdEHbBPAosSUsfqFH6/jtjdlFVF230
Q+u364GoD/NR6Em8tv3CAryQONrUKjQbHWxKhgKOHcECgYEA7Ep73QxlEzUpOIsy
k2aytjUxVU57sQeTA6igpRIgxvab8DL17WTXz/iwfDekqPzisfHJ9v51ggwv8PhS
pvgzDbsRP/emz0Z1UlVNukrcW6ZsoQGvCrQPoVGdsjwtsERff/GS9d4STfTQQSWL
Oi0tvdD7kalz+gtYSxY5GjNT18kCgYEA1SOaXNPnczKZvZu8a/H5H6J1DjxusPkX
4LZ8sN3bmrZDc4NYs/d99WLzTN0RF8sI03B3zrZm9cI1eE6pOIKcYIRFHGk2bxNj
zV2vhkpmFpEuFlTqyoRbcE1nvHX6V6ZUx3vvvMSbicaFQ13ZhjqRoj/TD0kL7ZlC
9jwGnhx046MCgYEAparPTztKfn4OSZumuRwO/psq3JGrPYJ++9i10SZ1nqn2ySEh
tfC3MxQ8wMrOgsDTPEm2/ZqIzsY2sq+YW4K3YNAglwXOiZLv3Or8FTo5Z3S2wugI
TuvR7ZvogbeZnPVDM9Qu4n1xvgCAJrzo8cANSwGD8Curqctce0C4hnsoNKkCgYEA
r3S1mAEhISXghcP0YnA5gp87+VIqVSlZTLUtBHQ+Waf88tSHau8sE5s3amj5rzqG
s3h8SADD1T/gwH8QsuJiVNnOAsth8iJmICMlYUlRrPYqmFujRL+cfmBaKzx7rzfP
xr/x5NV8rPhtr71MWkFQrd4Yoxag6SEnjIhxcis+1j0CgYEAziD8DUhIler3rWKD
LauBB0ZLBAyzA0W0t8Eb01y7+ROGB04k7kDWkKkkJGGT8JeyA9Be0FQ8bKsnKhbo
6dKMJq8xwBiDS3vrJyNEzPHPnThs5zDJdeBZrn6qjnSERfDYRnm9CDJuOP5+VsdQ
c9TCEaBGVFyH5ZMhzcSvLq8k8bU=
-----END PRIVATE KEY-----`,
							},
						},
					},
				},
				Services:        make(map[string]*apisix.Service),
				PluginMetadatas: make(map[string]*apisix.PluginMetadata),
				SSLs:            make(map[string]*apisix.SSL),
			},
		},

		{
			title:      "convert cert with nodes succ",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
									Host:     "a.b.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
								{
									Host:     "a.a.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost:  "node",
							TLSEnable: true,
						},
					},
				},
			},
			outConfig: &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"gateway.stage.test-resource": {
						Route: apisixv1.Route{
							Metadata: apisixv1.Metadata{
								ID:   "gateway.stage.test-resource",
								Name: "test-resource",
								Desc: "test resource",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "stage",
								},
							},
							Host:    "test.exmaple.com",
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
							Type: utils.StringPtr("roundrobin"),
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "a.b.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
								{
									Host:     "a.a.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost: func() *string { ret := "node"; return &ret }(),
							TLS: &apisix.UpstreamTLS{
								ClientCert: `-----BEGIN CERTIFICATE-----
MIIC1TCCAnugAwIBAgIQNz/VdMR2eGXtflpl2muEdzAKBggqhkjOPQQDAjAeMRww
GgYDVQQDExNiY3MtY2EtYmNzLXNlcnZpY2VzMB4XDTIyMDUxMTA4MjEwM1oXDTMw
MDcyODA4MjEwM1owcTELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkdEMQswCQYDVQQH
EwJTWjENMAsGA1UECRMEWUhKRDEQMA4GA1UEChMHVGVuY2VudDERMA8GA1UECxMI
Qmx1ZUtpbmcxFDASBgNVBAMTC2V4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAxLrVuFdnTVM9NxL9e9xRAWy/U8QT/huRbXAvpVySu2nd
TmCscGJbxxInAln24a8GIH0ZjWzv56VuFUmjZNvNkKqdwYZfc5iPUxAJ8gTOdb5z
JCZYMq/VMIzcprjmwSz9ojxGl/8ebiRWe0JnonTnVAqE1c1AYxiSiiefMopKdCc+
jg4C+rMhjwT+DSn5Y9kE35rszLHzL+7z26cLTZCrllzmujxJOz1mKJjka6koLx1e
F8qdye1BnuiZB8U60CpoG4qblGhGAZXdbWfYO7TbQ8vfqSbdSR4dxBzRDsReTqwZ
BEUPKWECsHrAMZdf+UxzHMi9P4irL1CfLujXyc2f+wIDAQABo30wezAOBgNVHQ8B
Af8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMAwGA1UdEwEB
/wQCMAAwHwYDVR0jBBgwFoAUAtcijTsbN0RkAIVDIGPPitXomEkwGwYDVR0RBBQw
EoIHYS5hLmNvbYIHKi5iLmNvbTAKBggqhkjOPQQDAgNIADBFAiA5t8qB9lSBpdA1
f8xjagaJjK1ec0P5euultTq1XLt4BwIhAJndkGiTFDshH9WntNCLltLUQDPaDqM0
gNClQrlM7gLP
-----END CERTIFICATE-----`,
								ClientKey: `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDEutW4V2dNUz03
Ev173FEBbL9TxBP+G5FtcC+lXJK7ad1OYKxwYlvHEicCWfbhrwYgfRmNbO/npW4V
SaNk282Qqp3Bhl9zmI9TEAnyBM51vnMkJlgyr9UwjNymuObBLP2iPEaX/x5uJFZ7
QmeidOdUCoTVzUBjGJKKJ58yikp0Jz6ODgL6syGPBP4NKflj2QTfmuzMsfMv7vPb
pwtNkKuWXOa6PEk7PWYomORrqSgvHV4Xyp3J7UGe6JkHxTrQKmgbipuUaEYBld1t
Z9g7tNtDy9+pJt1JHh3EHNEOxF5OrBkERQ8pYQKwesAxl1/5THMcyL0/iKsvUJ8u
6NfJzZ/7AgMBAAECggEBAKIiSRlQD3cO7xiAsiBuhuRht51VsBRwq/5Bw0LJdLS4
nweFbRiCN5ltQHETrAB7utTzxSdlbKLBGGS698qbzGM5+iIQmIIwbY7LXSb1ByLK
/yH/6Bh+CXml3gQZxzPV3ILkolmKjI3BrPSQ2dBuAGim8qsyKaqCCeOKnA2PI6Vt
cB30nO0BO0WziccZPBIR/vzTcHuh+em2cfz1kV+6Yt/n7R5Ds2JmC4RfTVTv7XyC
/FNAoLVbzLfoJmcAjCFghSQSezQDlR+M0eDdEHbBPAosSUsfqFH6/jtjdlFVF230
Q+u364GoD/NR6Em8tv3CAryQONrUKjQbHWxKhgKOHcECgYEA7Ep73QxlEzUpOIsy
k2aytjUxVU57sQeTA6igpRIgxvab8DL17WTXz/iwfDekqPzisfHJ9v51ggwv8PhS
pvgzDbsRP/emz0Z1UlVNukrcW6ZsoQGvCrQPoVGdsjwtsERff/GS9d4STfTQQSWL
Oi0tvdD7kalz+gtYSxY5GjNT18kCgYEA1SOaXNPnczKZvZu8a/H5H6J1DjxusPkX
4LZ8sN3bmrZDc4NYs/d99WLzTN0RF8sI03B3zrZm9cI1eE6pOIKcYIRFHGk2bxNj
zV2vhkpmFpEuFlTqyoRbcE1nvHX6V6ZUx3vvvMSbicaFQ13ZhjqRoj/TD0kL7ZlC
9jwGnhx046MCgYEAparPTztKfn4OSZumuRwO/psq3JGrPYJ++9i10SZ1nqn2ySEh
tfC3MxQ8wMrOgsDTPEm2/ZqIzsY2sq+YW4K3YNAglwXOiZLv3Or8FTo5Z3S2wugI
TuvR7ZvogbeZnPVDM9Qu4n1xvgCAJrzo8cANSwGD8Curqctce0C4hnsoNKkCgYEA
r3S1mAEhISXghcP0YnA5gp87+VIqVSlZTLUtBHQ+Waf88tSHau8sE5s3amj5rzqG
s3h8SADD1T/gwH8QsuJiVNnOAsth8iJmICMlYUlRrPYqmFujRL+cfmBaKzx7rzfP
xr/x5NV8rPhtr71MWkFQrd4Yoxag6SEnjIhxcis+1j0CgYEAziD8DUhIler3rWKD
LauBB0ZLBAyzA0W0t8Eb01y7+ROGB04k7kDWkKkkJGGT8JeyA9Be0FQ8bKsnKhbo
6dKMJq8xwBiDS3vrJyNEzPHPnThs5zDJdeBZrn6qjnSERfDYRnm9CDJuOP5+VsdQ
c9TCEaBGVFyH5ZMhzcSvLq8k8bU=
-----END PRIVATE KEY-----`,
							},
						},
					},
				},
				Services:        make(map[string]*apisix.Service),
				PluginMetadatas: make(map[string]*apisix.PluginMetadata),
				SSLs:            make(map[string]*apisix.SSL),
			},
		},

		{
			hasErr:     true,
			title:      "convert cert with nodes not exists",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
									Host:     "b.a.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost:  "node",
							TLSEnable: true,
						},
					},
				},
			},
			outConfig: nil,
		},

		{
			hasErr:     true,
			title:      "convert cert with nodes cert not exists",
			inputStage: getStage(),
			inputResources: []*v1beta1.BkGatewayResource{
				{
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
									Host:     "a.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
								{
									Host:     "a.a.com",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
							PassHost:  "node",
							TLSEnable: true,
						},
					},
				},
			},
			outConfig: nil,
		},
	}
	tree := getRadixTree(t)
	for index, test := range testCases {
		t.Logf("test %d: %s", index, test.title)
		tmpConverter, _ := NewConverter("", "gateway", test.inputStage, &UpstreamConfig{
			CertDetectTree: tree,
		}, nil)
		apisixConf, err := tmpConverter.Convert(context.Background(), test.inputResources, test.inputServices, nil, nil)
		assert.Equal(t, test.hasErr, err != nil, "err: (%+v), apisixConf: (%s)", err, marshallIgnoreErr(apisixConf))
		if test.hasErr {
			continue
		}
		assert.Equal(t, test.outConfig, apisixConf)
	}
}

func TestConverter_getResourceName(t *testing.T) {
	type args struct {
		specName string
		labels   map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test with spec name",
			args: args{
				specName: "test-spec-name",
				labels: map[string]string{
					config.BKAPIGatewayLabelKeyResourceName: "test-resource",
				},
			},
			want:    "test-spec-name",
			wantErr: false,
		},
		{
			name: "test with labels name",
			args: args{
				specName: "",
				labels: map[string]string{
					config.BKAPIGatewayLabelKeyResourceName: "test-resource",
				},
			},
			want:    "test-resource",
			wantErr: false,
		},
		{
			name: "test with labels name",
			args: args{
				specName: "",
				labels:   map[string]string{},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Converter{}
			got, err := c.getResourceName(tt.args.specName, tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("Converter.getResourceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Converter.getResourceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
