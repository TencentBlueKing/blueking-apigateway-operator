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

// Package cert provides the functionality to fetch TLS certificates from Kubernetes secrets.
package cert

import (
	"context"

	"github.com/rotisserie/eris"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// TLSCertFetcher ...
type TLSCertFetcher interface {
	GetTLSCertFromSecret(
		ctx context.Context,
		gatewayName, stageName, secretRef, namespace string,
	) (*v1beta1.TLSCert, error)
}

// RegistryTLSCertFetcher ...
type RegistryTLSCertFetcher struct {
	resourceRegistry registry.Registry
}

// NewRegistryTLSCertFetcher ...
func NewRegistryTLSCertFetcher(resourceRegistry registry.Registry) TLSCertFetcher {
	return &RegistryTLSCertFetcher{
		resourceRegistry: resourceRegistry,
	}
}

// GetTLSCertFromSecret ...
func (f *RegistryTLSCertFetcher) GetTLSCertFromSecret(
	ctx context.Context,
	gatewayName, stageName, secretRef, namespace string,
) (*v1beta1.TLSCert, error) {
	if secretRef == "" {
		return nil, eris.Errorf("No secret reference provided")
	}
	secret := corev1.Secret{}
	err := f.resourceRegistry.Get(
		ctx,
		registry.ResourceKey{
			StageInfo:    registry.StageInfo{GatewayName: gatewayName, StageName: stageName},
			ResourceName: secretRef,
		},
		&secret,
	)
	if err != nil {
		return nil, err
	}

	tlsCert, err := v1beta1.GetTLSCertFromSecret(&secret)
	if err != nil {
		return nil, eris.Errorf(
			"Get cert from secret(%s/%s) failed: no 'tls.crt' or 'tls.key' fields",
			namespace,
			secretRef,
		)
	}

	return tlsCert, nil
}
