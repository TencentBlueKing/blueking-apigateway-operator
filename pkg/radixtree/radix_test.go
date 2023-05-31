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

package radixtree

import (
	"fmt"
	"strconv"
	"testing"

	v1beta1 "micro-gateway/api/v1beta1"
	"micro-gateway/pkg/logging"
	"micro-gateway/pkg/registry"

	"github.com/armon/go-radix"
	"github.com/stretchr/testify/assert"
)

func TestReverse(t *testing.T) {
	tree := &SuffixRadixTree{}
	assert.Equal(t, "abc", tree.reverse("cba"))
	assert.Equal(t, "c", tree.reverse("c"))
	assert.Equal(t, "ab", tree.reverse("ba"))
	assert.Equal(t, "abcd", tree.reverse("dcba"))
	assert.Equal(t, "", tree.reverse(""))
}

func TestReverseSNI(t *testing.T) {
	tree := &SuffixRadixTree{}
	assert.Equal(t, "moc.a$", tree.buildReverseMatchPath("a.com"))
	assert.Equal(t, "moc.a.", tree.buildReverseMatchPath("*.a.com"))
	assert.Equal(t, "moc.a", tree.buildReverseMatchPath("*a.com"))
	assert.Equal(t, "moc.c.b.a$", tree.buildReverseMatchPath("a.b.c.com"))
	assert.Equal(t, "$", tree.buildReverseMatchPath(""))
}

func buildInsertedTree(t *testing.T) *SuffixRadixTree {
	tree := &SuffixRadixTree{
		secretMap: map[registry.ResourceKey]*v1beta1.TLSCert{
			{
				StageInfo: registry.StageInfo{
					GatewayName: "testgateway",
					StageName:   "teststage",
				},
				ResourceName: "testname",
			}: {
				Cert:   "testcert",
				Key:    "testkey",
				CACert: "testca",
			},
		},
		pathDocumentMap: make(map[string]map[registry.ResourceKey]struct{}),
		radixTree:       radix.New(),
		logger:          logging.GetLogger(),
	}
	tree.insertSNIs(registry.ResourceKey{
		StageInfo: registry.StageInfo{
			GatewayName: "testgateway",
			StageName:   "teststage",
		},
		ResourceName: "testname",
	}, []string{"a.com", "*.b.com"})
	return tree
}

func TestInsertSNIs(t *testing.T) {
	tree := buildInsertedTree(t)
	assert.Equal(t, 1, len(tree.pathDocumentMap["a.com"]))
	assert.Equal(t, 1, len(tree.pathDocumentMap["*.b.com"]))

	prefix, cert, matched := tree.MatchLongestPrefixWithRandomCert("a.com")
	assert.Equal(t, "moc.a$", prefix)
	assert.Equal(t, v1beta1.TLSCert{
		Cert:   "testcert",
		Key:    "testkey",
		CACert: "testca",
	}, *cert)
	assert.Equal(t, true, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("b.com")
	var nilCert *v1beta1.TLSCert
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.b.com")
	assert.Equal(t, "moc.b.", prefix)
	assert.Equal(t, v1beta1.TLSCert{
		Cert:   "testcert",
		Key:    "testkey",
		CACert: "testca",
	}, *cert)
	assert.Equal(t, true, matched)
}

func TestRemoveSNIs(t *testing.T) {
	tree := buildInsertedTree(t)
	tree.removeSNIs(registry.ResourceKey{
		StageInfo: registry.StageInfo{
			GatewayName: "testgateway",
			StageName:   "teststage",
		},
		ResourceName: "testname",
	}, []string{"a.com", "b.com"})
	assert.Equal(t, 0, len(tree.pathDocumentMap["a.com"]))
	assert.Equal(t, 0, len(tree.pathDocumentMap["b.com"]))
	assert.Equal(t, 1, len(tree.pathDocumentMap["*.b.com"]))

	var nilCert *v1beta1.TLSCert
	prefix, cert, matched := tree.MatchLongestPrefixWithRandomCert("a.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("b.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.b.com")
	assert.Equal(t, "moc.b.", prefix)
	assert.Equal(t, v1beta1.TLSCert{
		Cert:   "testcert",
		Key:    "testkey",
		CACert: "testca",
	}, *cert)
	assert.Equal(t, true, matched)
}

var obj1 = registry.ResourceKey{
	StageInfo: registry.StageInfo{
		GatewayName: "testgateway",
		StageName:   "teststage",
	},
	ResourceName: "obj1",
}

var obj2 = registry.ResourceKey{
	StageInfo: registry.StageInfo{
		GatewayName: "testgateway",
		StageName:   "teststage",
	},
	ResourceName: "obj2",
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

// a.com, *.c.com
var cert3 = v1beta1.TLSCert{
	Cert: `-----BEGIN CERTIFICATE-----
MIIC1DCCAnqgAwIBAgIRAIrms30PYvdcou1vcANgixswCgYIKoZIzj0EAwIwHjEc
MBoGA1UEAxMTYmNzLWNhLWJjcy1zZXJ2aWNlczAeFw0yMjA1MTEwODIyMThaFw0z
MDA3MjgwODIyMThaMHExCzAJBgNVBAYTAkNOMQswCQYDVQQIEwJHRDELMAkGA1UE
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
MBCCBWEuY29tggcqLmMuY29tMAoGCCqGSM49BAMCA0gAMEUCIQD2MA7Kp2e169fY
4pD4ifyUCLFtKmnX82qpCy/rzsIoHgIgbOM8gjO8bBse/K5rWojh/azL8BjNwPUI
asVnKgC0PAE=
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

func TestInsert(t *testing.T) {
	tree := NewSuffixRadixTree()
	var nilErr error
	// a.com, *.b.com
	ok, err := tree.Insert(obj1, &cert1)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)

	prefix, cert, matched := tree.MatchLongestPrefixWithRandomCert("a.com")
	assert.Equal(t, "moc.a$", prefix)
	assert.Equal(t, cert1, *cert)
	assert.Equal(t, true, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("b.com")
	var nilCert *v1beta1.TLSCert
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.b.com")
	assert.Equal(t, "moc.b.", prefix)
	assert.Equal(t, cert1, *cert)
	assert.Equal(t, true, matched)

	// a.com, *.c.com
	ok, err = tree.Insert(obj2, &cert3)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.com")
	assert.Equal(t, "moc.a$", prefix)
	assert.Condition(t, func() (success bool) {
		return assert.ObjectsAreEqual(cert1, *cert) || assert.ObjectsAreEqualValues(cert3, *cert)
	}, "actual certs: %v", *cert)
	assert.Equal(t, true, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("c.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.c.com")
	assert.Equal(t, "moc.c.", prefix)
	assert.Equal(t, cert3, *cert)
	assert.Equal(t, true, matched)
}

func TestUpdate(t *testing.T) {
	tree := NewSuffixRadixTree()
	var nilErr error
	var nilCert *v1beta1.TLSCert
	// a.com, *.b.com
	ok, err := tree.Insert(obj1, &cert1)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)
	// a.a.com, *.b.com
	ok, err = tree.Update(obj1, &cert2)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)

	prefix, cert, matched := tree.MatchLongestPrefixWithRandomCert("a.a.a.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.a.com")
	assert.Equal(t, "moc.a.a$", prefix)
	assert.Equal(t, cert2, *cert)
	assert.Equal(t, true, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("b.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.b.com")
	assert.Equal(t, "moc.b.", prefix)
	assert.Equal(t, cert2, *cert)
	assert.Equal(t, true, matched)
}

func TestDelete(t *testing.T) {
	tree := NewSuffixRadixTree()
	var nilErr error
	var nilCert *v1beta1.TLSCert
	// a.com, *.b.com
	ok, err := tree.Insert(obj1, &cert1)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)
	// a.com, *.c.com
	ok, err = tree.Insert(obj2, &cert3)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)
	// a.com, *.c.com
	ok, err = tree.Delete(obj2)
	assert.Equal(t, nilErr, err)
	assert.Equal(t, true, ok)

	prefix, cert, matched := tree.MatchLongestPrefixWithRandomCert("a.com")
	assert.Equal(t, "moc.a$", prefix)
	assert.Equal(t, cert1, *cert)
	assert.Equal(t, true, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("b.com")
	assert.Equal(t, "", prefix)
	assert.Equal(t, nilCert, cert)
	assert.Equal(t, false, matched)

	prefix, cert, matched = tree.MatchLongestPrefixWithRandomCert("a.b.com")
	assert.Equal(t, "moc.b.", prefix)
	assert.Equal(t, cert1, *cert)
	assert.Equal(t, true, matched)
}

// Benchmarsk

func BenchmarkRadixTreeInsert(b *testing.B) {
	tree := &SuffixRadixTree{
		secretMap:       make(map[registry.ResourceKey]*v1beta1.TLSCert),
		pathDocumentMap: make(map[string]map[registry.ResourceKey]struct{}),
		radixTree:       radix.New(),
	}
	for n := 0; n < b.N; n++ {
		obj := registry.ResourceKey{ResourceName: strconv.Itoa(n)}
		tree.secretMap[obj] = &v1beta1.TLSCert{
			Cert:   "testcert",
			Key:    "testkey",
			CACert: "testca",
		}
		tree.insertSNIs(obj, []string{fmt.Sprintf("*.x.y.z.%d", n)})
	}
}

func BenchmarkRadixTreeSearch(b *testing.B) {
	tree := &SuffixRadixTree{
		secretMap:       make(map[registry.ResourceKey]*v1beta1.TLSCert),
		pathDocumentMap: make(map[string]map[registry.ResourceKey]struct{}),
		radixTree:       radix.New(),
	}
	for n := 0; n < b.N; n++ {
		obj := registry.ResourceKey{ResourceName: strconv.Itoa(n)}
		tree.secretMap[obj] = &v1beta1.TLSCert{
			Cert:   "testcert",
			Key:    "testkey",
			CACert: "testca",
		}
		tree.insertSNIs(obj, []string{fmt.Sprintf("*.x.y.z.%d", n)})
	}
	host := fmt.Sprintf("a.x.y.z.%d", b.N-1)
	for n := 0; n < b.N; n++ {
		tree.MatchLongestPrefixWithRandomCert(host)
	}
}
