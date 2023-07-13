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
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/rotisserie/eris"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/utils"
)

// Status is service discovery's status
type Status int

const (
	innerStatusIdle Status = iota
	innerStatusRunning
)

// ServiceDiscovery is interface of BkGatewayService discovery
type ServiceDiscovery interface {
	Apply(*Service) error
	StopDiscovery()
	Clean()
}

// RegistryConfig is config for each updation of external discovery config
type RegistryConfig struct {
	gatewaySvc      *Service
	discoveryConfig map[string]interface{}
	// lastFailedEndpoints for retry, nil means last updation is successful
	lastFailedEndpoints *gatewayv1beta1.BkGatewayEndpointsSpec
	// determine whether watch/list goroutine has exited
	innerStatus Status
	// context for watch/list goroutine
	registryCTX    context.Context
	registryCancle context.CancelFunc
}

// ServiceDiscoveryImpl is implementation for ServiceDiscovery
type ServiceDiscoveryImpl struct {
	sync.RWMutex
	// context for service update endpoints retry goroutine
	ctx    context.Context
	cancel context.CancelFunc

	// config for each external discovery config updating
	config *RegistryConfig
	// user's registry interface
	registry types.Registry
	logger   logr.Logger
	svcKey   string
	client   client.Client
}

// NewServiceDiscovery return a ServiceDiscovery for service `svcKey`
func NewServiceDiscovery(client client.Client, registry types.Registry, svcKey string) ServiceDiscovery {
	retSvc := &ServiceDiscoveryImpl{
		registry: registry,
		svcKey:   svcKey,
		logger:   ctrl.Log.WithName("ServiceDiscovery").WithValues("svcKey", svcKey, "registryName", registry.Name()),
		config:   &RegistryConfig{},
		client:   client,
	}
	retSvc.ctx, retSvc.cancel = context.WithCancel(context.Background())
	go retSvc.retryUpdateGoroutine()
	return retSvc
}

// Apply apply the service config changes
func (sd *ServiceDiscoveryImpl) Apply(svc *Service) error {
	// diff config
	if svc == nil {
		sd.logger.Error(nil, "Apply the empty service")
		return eris.Errorf("Apply the empty service for serviceDiscovery of svcKey(%s)", sd.svcKey)
	}
	newConfig, err := utils.RawExtension2Map(svc.Upstream.ExternalDiscoveryConfig)
	if err != nil {
		sd.logger.Error(err, "ExternalDiscoveryConfig convert failed", "config", svc.Upstream.ExternalDiscoveryConfig)
		return err
	}
	if sd.config.gatewaySvc != nil &&
		sd.config.gatewaySvc.Upstream.ServiceName == svc.Upstream.ServiceName {
		if reflect.DeepEqual(sd.config.discoveryConfig, newConfig) {
			sd.logger.Info(
				"Config of new service is same as old one, skip applying",
				"config",
				svc.Upstream.ExternalDiscoveryConfig,
			)
			return nil
		}
	}
	// update service discovery config
	sd.logger.Info("Use new config to start service discovery", "config", svc.Upstream.ExternalDiscoveryConfig)
	if sd.config.gatewaySvc != nil {
		sd.StopDiscovery()
	}
	sd.Lock()
	sd.config = &RegistryConfig{
		gatewaySvc:      svc,
		discoveryConfig: newConfig,
	}
	sd.Unlock()

	sd.config.registryCTX, sd.config.registryCancle = context.WithCancel(context.Background())
	supportMethods := sd.registry.DiscoveryMethods()
	switch supportMethods {
	case types.WatchSupported:
		sd.config.innerStatus = innerStatusRunning
		go sd.startWatch(sd.config)
	case types.ListSupported:
		sd.config.innerStatus = innerStatusRunning
		go sd.startList(sd.config)
	case types.WatchAndListSupported:
		sd.config.innerStatus = innerStatusRunning
		go sd.startWatch(sd.config)
		go sd.periodlyListSync(sd.config, time.Minute)
	default:
		sd.logger.Error(
			nil,
			"Can not recognize registry's discovery methods",
			"supportMethods",
			sd.registry.DiscoveryMethods(),
		)
		return eris.Errorf(
			"Can not recognize registry's discovery methods. supportMethods: %v",
			sd.registry.DiscoveryMethods(),
		)
	}
	return nil
}

// StopDiscovery stop the discovery
//
//nolint:gosimple
func (sd *ServiceDiscoveryImpl) StopDiscovery() {
	sd.config.registryCancle()

	// check whether Watch or List function stopped
	go func(config *RegistryConfig) {
		checkTicker := time.NewTicker(time.Minute)
		defer checkTicker.Stop()
		for {
			select {
			case <-checkTicker.C:
				if config.innerStatus == innerStatusIdle {
					sd.logger.V(1).
						Info("Registry List/Watch goroutine stopped successfully with old config", "config", config.discoveryConfig)
					return
				}
				sd.logger.Error(
					nil,
					"Registry List/Watch goroutine does not stop with old config, watching out for memory leaks",
					"config",
					config.discoveryConfig,
				)
			}
		}
	}(sd.config)
}

// Clean stop the discovery, stop the retry goroutine and delete endpoints of service
func (sd *ServiceDiscoveryImpl) Clean() {
	sd.StopDiscovery()
	sd.cancel()
	sd.deleteEndpoints()
}

func (sd *ServiceDiscoveryImpl) startWatch(config *RegistryConfig) {
	defer func() {
		config.innerStatus = innerStatusIdle
	}()

	// registry call back function, if watch goroutine should exit, call back will not response to the watch events
	callBackFunc := func(endpoints *gatewayv1beta1.BkGatewayEndpointsSpec) error {
		if config.registryCTX.Err() != nil {
			sd.logger.Error(config.registryCTX.Err(), "CallBack function is called after the watch channel is closed")
			return config.registryCTX.Err()
		}
		sd.Lock()
		err := sd.updateEndpoints(endpoints)
		sd.Unlock()
		return err
	}

	for {
		if config.registryCTX.Err() != nil {
			sd.logger.Info(
				"Registry watch channel closed, exit watch with old discovery config",
				"config",
				config.discoveryConfig,
			)
			return
		}

		err := sd.registry.Watch(
			config.registryCTX,
			config.gatewaySvc.Upstream.ServiceName,
			config.gatewaySvc.Namespace,
			config.discoveryConfig,
			callBackFunc,
		)
		if err != nil {
			sd.logger.Error(
				err,
				"Registry watch service endpoints failed, retry in 5 seconds",
				"config",
				config.discoveryConfig,
			)
			time.Sleep(time.Second * 5)
			continue
		}
		// return when watch is canceled
		sd.logger.Info(
			"Registry watch channel exit without error, stop watch with old discovery config",
			"config",
			config.discoveryConfig,
		)
		return
	}
}

func (sd *ServiceDiscoveryImpl) startList(config *RegistryConfig) {
	defer func() {
		config.innerStatus = innerStatusIdle
	}()

	sd.periodlyListSync(config, time.Second*15)
}

func (sd *ServiceDiscoveryImpl) periodlyListSync(config *RegistryConfig, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-config.registryCTX.Done():
			sd.logger.Info(
				"Registry context canceled, exit list with old discovery config",
				"config",
				config.discoveryConfig,
			)
			return
		case <-ticker.C:
			endpoints, err := sd.registry.List(
				config.gatewaySvc.Upstream.ServiceName,
				config.gatewaySvc.Namespace,
				config.discoveryConfig,
			)
			if err != nil {
				sd.logger.Error(
					err,
					"Registry list service endpoints failed, retry in next period",
					"config",
					config.discoveryConfig,
				)
				break
			}
			sd.Lock()
			err = sd.updateEndpoints(endpoints)
			sd.Unlock()
			if err != nil {
				sd.logger.Error(
					err,
					"ServiceDiscovery update Upstream for BkGatewayService failed, retry in next period",
					"config",
					config.discoveryConfig,
				)
				break
			}
		}
	}
}

func (sd *ServiceDiscoveryImpl) getGatewayEndpointsName() string {
	return fmt.Sprintf(
		"%s.%s.%s",
		sd.registry.Name(),
		types.KindMapping[sd.config.gatewaySvc.Kind],
		sd.config.gatewaySvc.Name,
	)
}

func (sd *ServiceDiscoveryImpl) updateEndpoints(spec *gatewayv1beta1.BkGatewayEndpointsSpec) error {
	if spec == nil || spec.Nodes == nil {
		spec = &gatewayv1beta1.BkGatewayEndpointsSpec{Nodes: make([]gatewayv1beta1.BkGatewayNode, 0)}
	}
	sd.config.lastFailedEndpoints = nil
	endpoints := &gatewayv1beta1.BkGatewayEndpoints{}
	endpoints.SetNamespace(sd.config.gatewaySvc.Namespace)
	endpoints.SetName(sd.getGatewayEndpointsName())
	// get old endpoints
	err := sd.client.Get(context.Background(), client.ObjectKeyFromObject(endpoints), endpoints)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			sd.logger.Info("GatewayName endpoints not found, create one")
			newEndpoints := &gatewayv1beta1.BkGatewayEndpoints{}
			newEndpoints.SetNamespace(sd.config.gatewaySvc.Namespace)
			newEndpoints.SetName(sd.getGatewayEndpointsName())
			newEndpoints.Labels = sd.setLabels(newEndpoints.Labels)
			spec.DeepCopyInto(&newEndpoints.Spec)
			innerErr := sd.client.Create(context.Background(), newEndpoints)
			if innerErr != nil {
				sd.logger.Error(innerErr, "Create gateway endpoints failed", "endpoints", *newEndpoints)
				sd.config.lastFailedEndpoints = spec
				return innerErr
			}
			return nil
		}
		sd.logger.Error(err, "Get gateway endpoints from apiserver failed, skip endpoints diff")
	}

	sd.logger.V(1).Info("Nodes diff", "new nodes", spec.Nodes, "old nodes", endpoints.Spec.Nodes)

	// diff nodes
	needUpdate := false
	newNodesMap := make(map[string]gatewayv1beta1.BkGatewayNode)
	oldNodesMap := make(map[string]gatewayv1beta1.BkGatewayNode)
	for _, node := range spec.Nodes {
		newNodesMap[fmt.Sprintf("%s:%d", node.Host, node.Port)] = node
	}
	for _, node := range endpoints.Spec.Nodes {
		oldNodesMap[fmt.Sprintf("%s:%d", node.Host, node.Port)] = node
	}

	for host, node := range oldNodesMap {
		if newNode, ok := newNodesMap[host]; !ok || !reflect.DeepEqual(node, newNode) {
			needUpdate = true
			break
		}
		delete(newNodesMap, host)
	}
	if !needUpdate &&
		(len(newNodesMap) != 0 || sd.keyLabelChanged(endpoints.Labels)) {
		needUpdate = true
	}
	if !needUpdate {
		sd.logger.V(1).Info("New endpoints is same as old endpoints, do not need update")
		return nil
	}

	// update endpoints
	spec.DeepCopyInto(&endpoints.Spec)
	endpoints.Labels = sd.setLabels(endpoints.Labels)
	err = sd.client.Update(context.Background(), endpoints)
	if err != nil {
		sd.logger.Error(err, "Update gateway endpoints failed", "endpoints", endpoints)
		sd.config.lastFailedEndpoints = spec
		return err
	}
	sd.logger.V(1).Info("Update gateway endpoints succ", "endpoints", endpoints)
	return nil
}

func (sd *ServiceDiscoveryImpl) retryUpdateGoroutine() {
	retryTicker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-sd.ctx.Done():
			return
		case <-retryTicker.C:
			sd.Lock()
			eps := sd.config.lastFailedEndpoints
			if eps != nil {
				sd.logger.V(1).Info("retry update endpoints with last failed endpoints", "endpoints", *eps)
				sd.updateEndpoints(eps)
			}
			sd.Unlock()
		}
	}
}

func (sd *ServiceDiscoveryImpl) deleteEndpoints() error {
	endpoints := &gatewayv1beta1.BkGatewayEndpoints{}
	endpoints.SetNamespace(sd.config.gatewaySvc.Namespace)
	endpoints.SetName(sd.getGatewayEndpointsName())
	err := sd.client.Delete(context.Background(), endpoints)
	if err != nil {
		sd.logger.Error(err, "Delete gateway endpoints failed", "name", sd.getGatewayEndpointsName())
		return err
	}
	return nil
}

func (sd *ServiceDiscoveryImpl) setLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	if val, ok := labels[types.ManagedByLabelTag]; !ok || val != sd.registry.Name() {
		labels[types.ManagedByLabelTag] = sd.registry.Name()
	}
	labels[config.BKAPIGatewayLabelKeyGatewayName] = sd.config.gatewaySvc.Labels[config.BKAPIGatewayLabelKeyGatewayName]
	labels[config.BKAPIGatewayLabelKeyGatewayStage] = sd.config.gatewaySvc.Labels[config.BKAPIGatewayLabelKeyGatewayStage]
	return labels
}

func (sd *ServiceDiscoveryImpl) keyLabelChanged(labels map[string]string) bool {
	return !checkLabelSame(labels, sd.config.gatewaySvc.Labels, config.BKAPIGatewayLabelKeyGatewayName) ||
		!checkLabelSame(labels, sd.config.gatewaySvc.Labels, config.BKAPIGatewayLabelKeyGatewayStage) ||
		labels[types.ManagedByLabelTag] != sd.registry.Name()
}

func checkLabelSame(lhs, rhs map[string]string, key string) bool {
	return lhs[key] == rhs[key]
}
