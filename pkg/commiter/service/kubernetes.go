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

package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

/*

查询指定service对应的endpoints, 并组装成v1beta1.BkGatewayNode对象返回

用于专享网关的服务发现机制

*/

const (
	serviceSpliter     = "/"
	servicePortSpliter = ":"
)

// NodeDiscoverer ...
type NodeDiscoverer interface {
	GetNodes(workNamespace, serviceName string) ([]v1beta1.BkGatewayNode, error)
}

// KubernetesNodeDiscoverer converts kubernetes service to upstream node
type KubernetesNodeDiscoverer struct {
	kubeClient client.Client

	logger *zap.SugaredLogger
}

// NewKubernetesNodeDiscoverer create new kubernetes converter
func NewKubernetesNodeDiscoverer(kubeClient client.Client) NodeDiscoverer {
	if kubeClient == nil {
		return nil
	}

	return &KubernetesNodeDiscoverer{
		kubeClient: kubeClient,
		logger:     logging.GetLogger().Named("kubernetes-node-discoverer"),
	}
}

// GetNodes get upstream nodes according to service name
func (kd *KubernetesNodeDiscoverer) GetNodes(workNamespace, serviceName string) ([]v1beta1.BkGatewayNode, error) {
	svcName, svcNamespace, port, err := kd.getSvcInfo(workNamespace, serviceName)
	if err != nil {
		return nil, err
	}
	svc := &k8scorev1.Service{}
	if err := kd.kubeClient.Get(context.TODO(), k8stypes.NamespacedName{
		Name:      svcName,
		Namespace: svcNamespace,
	}, svc); err != nil {
		if k8serrors.IsNotFound(err) {
			kd.logger.Info(fmt.Sprintf("service %s not found", serviceName))
			return nil, nil
		}
		return nil, eris.Wrapf(err, "get service %s/%s failed", svcName, svcNamespace)
	}
	foundSvcPort := false
	var svcPort *k8scorev1.ServicePort
	for _, tmpPort := range svc.Spec.Ports {
		if tmpPort.Port == int32(port) {
			svcPort = &tmpPort
			foundSvcPort = true
			break
		}
	}
	if !foundSvcPort {
		kd.logger.Info(fmt.Sprintf("port of service %s not found", serviceName))
		return nil, nil
	}
	endpoints := &k8scorev1.Endpoints{}
	if err := kd.kubeClient.Get(context.TODO(), k8stypes.NamespacedName{
		Name:      svcName,
		Namespace: svcNamespace,
	}, endpoints); err != nil {
		return nil, eris.Wrapf(err, "get endpoints %s/%s failed", svcName, svcNamespace)
	}
	var nodes []v1beta1.BkGatewayNode
	for _, subset := range endpoints.Subsets {
		var targetPort int
		for _, subsetPort := range subset.Ports {
			if svcPort.TargetPort.String() == subsetPort.Name ||
				svcPort.TargetPort.IntValue() == int(subsetPort.Port) {
				targetPort = int(subsetPort.Port)

				break
			}
		}
		if targetPort == 0 {
			continue
		}
		for _, readyAddr := range subset.Addresses {
			nodes = append(nodes, v1beta1.BkGatewayNode{
				Host:   readyAddr.IP,
				Port:   targetPort,
				Weight: 10,
			})
		}
		for _, notReadyAddr := range subset.NotReadyAddresses {
			nodes = append(nodes, v1beta1.BkGatewayNode{
				Host:   notReadyAddr.IP,
				Port:   targetPort,
				Weight: 0,
			})
		}
	}
	return nodes, nil
}

func (kd *KubernetesNodeDiscoverer) getSvcInfo(defaultNs, serviceName string) (string, string, int, error) {
	svcStrs := strings.Split(serviceName, servicePortSpliter)
	if len(svcStrs) != 2 {
		return "", "", 0, eris.Errorf("invalid service name %s", serviceName)
	}
	svcNameNs := svcStrs[0]
	svcPortStr := svcStrs[1]
	svcPort, err := strconv.Atoi(svcPortStr)
	if err != nil {
		return "", "", 0, eris.Wrapf(err, "invalid service name %s, invalid port str %s",
			serviceName, svcPortStr)
	}
	var svcName string
	var svcNamespace string
	strs := strings.Split(svcNameNs, serviceSpliter)
	switch len(strs) {
	case 1:
		svcName = strs[0]
		svcNamespace = defaultNs
	case 2:
		svcName = strs[0]
		svcNamespace = strs[1]
	default:
		return "", "", 0, eris.Errorf("invalid service name %s", svcNameNs)
	}

	return svcName, svcNamespace, svcPort, nil
}
