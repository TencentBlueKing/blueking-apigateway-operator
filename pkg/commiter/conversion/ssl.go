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
	"crypto/x509"
	"encoding/pem"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"micro-gateway/api/v1beta1"
	"micro-gateway/pkg/apisix"
	"micro-gateway/pkg/utils"
)

// convertSSL convert ssl crd to apisix ssl object
func (c *Converter) convertSSL(ctx context.Context, ssl *v1beta1.BkGatewayTLS) (*apisix.SSL, error) {
	cert, err := c.sslConfig.CertFetcher.GetTLSCertFromSecret(
		ctx,
		c.gatewayName,
		c.stageName,
		ssl.Spec.GatewayTLSSecretRef,
		ssl.Namespace,
	)
	if err != nil {
		c.logger.Error(err, "Get TLSCert failed")
		return nil, err
	}

	snis := make([]string, 0)
	if len(ssl.Spec.SNIs) == 0 {
		pemdata := []byte(cert.Cert)
		for {
			pb, rest := pem.Decode(pemdata)
			if pb == nil {
				break
			}
			pemdata = rest
			certs, err := x509.ParseCertificates(pb.Bytes)
			if err != nil {
				c.logger.Error(err, "Parse Certificate from pem formatted cert failed")
				continue
			}
			for _, cert := range certs {
				// build domain
				domains := make(map[string]struct{})
				for _, dns := range cert.DNSNames {
					domains[dns] = struct{}{}
				}
				for _, uri := range cert.URIs {
					domains[uri.Host] = struct{}{}
				}
				for _, ip := range cert.IPAddresses {
					domains[ip.String()] = struct{}{}
				}
				domains[cert.Subject.CommonName] = struct{}{}

				// check domains
				for domain := range domains {
					if utils.HostMatch(domain) {
						snis = append(snis, domain)
					} else {
						c.logger.Infow("SSL's domain does not match required pattern", "domain", domain)
					}
				}
			}
		}
	} else {
		snis = append(snis, ssl.Spec.SNIs...)
	}

	apisixSSL := &apisix.SSL{
		Ssl: v1.Ssl{
			ID:     c.getID(ssl.Spec.ID, getObjectName(ssl.GetName(), ssl.GetNamespace())),
			Labels: c.getLabel(),
			Snis:   snis,
			Cert:   cert.Cert,
			Key:    cert.Key,
		},
	}

	c.logger.Debugw("convert ssl crd to apisix ssl object", "ssl", ssl, "apisixSSL", apisixSSL)

	return apisixSSL, nil
}
