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

package etcd

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

var apisixResourceTypes = []string{
	ApisixResourceTypeRoutes,
	ApisixResourceTypeStreamRoutes,
	ApisixResourceTypeServices,
	ApisixResourceTypeSSL,
	ApisixResourceTypePluginMetadata,
}

// EtcdConfigStore ...
type EtcdConfigStore struct {
	client *clientv3.Client
	prefix string

	stores map[string]*resourceStore
	differ *configDiffer

	logger *zap.SugaredLogger

	putInterval time.Duration

	lock *sync.RWMutex
}

// NewEtcdConfigStore ...
func NewEtcdConfigStore(client *clientv3.Client, prefix string, putInterval time.Duration) (*EtcdConfigStore, error) {
	s := &EtcdConfigStore{
		client:      client,
		prefix:      strings.TrimRight(prefix, "/"),
		stores:      make(map[string]*resourceStore, 4),
		differ:      newConfigDiffer(),
		logger:      logging.GetLogger().Named("etcd-config-store"),
		putInterval: putInterval,
		lock:        &sync.RWMutex{},
	}
	s.Init()

	s.logger.Infow("Create etcd config store", "prefix", prefix)

	if len(s.stores) != len(apisixResourceTypes) {
		s.logger.Error("Create etcd config store failed")
		return nil, fmt.Errorf("create etcd config store failed")
	}

	return s, nil
}

func (s *EtcdConfigStore) Init() {
	wg := &sync.WaitGroup{}
	for _, resourceType := range apisixResourceTypes {
		wg.Add(1)

		// 避免闭包导致变量覆盖问题
		tempResourceType := resourceType
		utils.GoroutineWithRecovery(context.Background(), func() {
			defer wg.Done()
			resourceStore, err := newResourceStore(s.client, s.prefix+"/"+tempResourceType+"/")
			if err != nil {
				s.logger.Errorw("Create resource store failed", "resourceType", tempResourceType)
				return
			}
			s.lock.Lock()
			defer s.lock.Unlock()
			s.stores[tempResourceType] = resourceStore
		})
	}
	wg.Wait()
}

// Get get a staged apisix configuration
func (s *EtcdConfigStore) Get(stageName string) *apisix.ApisixConfiguration {
	ret := apisix.NewEmptyApisixConfiguration()
	routes := s.stores[ApisixResourceTypeRoutes].getStageResources(stageName)
	for key, val := range routes {
		ret.Routes[key] = val.(*apisix.Route)
	}
	streamRoutes := s.stores[ApisixResourceTypeStreamRoutes].getStageResources(stageName)
	for key, val := range streamRoutes {
		ret.StreamRoutes[key] = val.(*apisix.StreamRoute)
	}
	services := s.stores[ApisixResourceTypeServices].getStageResources(stageName)
	for key, val := range services {
		ret.Services[key] = val.(*apisix.Service)
	}
	ssls := s.stores[ApisixResourceTypeSSL].getStageResources(stageName)
	for key, val := range ssls {
		ret.SSLs[key] = val.(*apisix.SSL)
	}
	pms := s.stores[ApisixResourceTypePluginMetadata].getStageResources(stageName)
	for key, val := range pms {
		ret.PluginMetadatas[key] = val.(*apisix.PluginMetadata)
	}
	return ret
}

// GetAll get staged apisix configuration map
func (s *EtcdConfigStore) GetAll() map[string]*apisix.ApisixConfiguration {
	configMap := make(map[string]*apisix.ApisixConfiguration)
	routeMap := s.stores[ApisixResourceTypeRoutes].getAllResources()
	for key, route := range routeMap {
		stageName := route.GetStageFromLabel()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = apisix.NewEmptyApisixConfiguration()
		}
		configMap[stageName].Routes[key] = route.(*apisix.Route)
	}

	streamRouteMap := s.stores[ApisixResourceTypeStreamRoutes].getAllResources()
	for key, route := range streamRouteMap {
		stageName := route.GetStageFromLabel()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = apisix.NewEmptyApisixConfiguration()
		}
		configMap[stageName].StreamRoutes[key] = route.(*apisix.StreamRoute)
	}

	serviceMap := s.stores[ApisixResourceTypeServices].getAllResources()
	for key, service := range serviceMap {
		stageName := service.GetStageFromLabel()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = apisix.NewEmptyApisixConfiguration()
		}
		configMap[stageName].Services[key] = service.(*apisix.Service)
	}

	sslMap := s.stores[ApisixResourceTypeSSL].getAllResources()
	for key, ssl := range sslMap {
		stageName := ssl.GetStageFromLabel()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = apisix.NewEmptyApisixConfiguration()
		}
		configMap[stageName].SSLs[key] = ssl.(*apisix.SSL)
	}

	pmMap := s.stores[ApisixResourceTypePluginMetadata].getAllResources()
	for key, pm := range pmMap {
		stageName := pm.GetStageFromLabel()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = apisix.NewEmptyApisixConfiguration()
		}
		configMap[stageName].PluginMetadatas[key] = pm.(*apisix.PluginMetadata)
	}

	return configMap
}

// Alter ...
func (s *EtcdConfigStore) Alter(
	ctx context.Context,
	stageName string,
	config *apisix.ApisixConfiguration,
) error {
	st := time.Now()
	err := s.alterByStage(ctx, stageName, config)

	// metric
	synchronizer.ReportStageConfigAlterMetric(stageName, config, st, err)

	if err != nil {
		s.logger.Errorw("Alter by stage failed", "err", err, "stage", stageName)
		return err
	}

	return nil
}

func (s *EtcdConfigStore) alterByStage(
	ctx context.Context, stageKey string, conf *apisix.ApisixConfiguration,
) (err error) {
	// get cached config
	oldConf := s.Get(stageKey)

	// diff config
	putConf, deleteConf := s.differ.diff(oldConf, conf)
	// put resources
	if putConf != nil {
		if err = s.batchPutResource(ctx, ApisixResourceTypeSSL, putConf.SSLs); err != nil {
			return fmt.Errorf("batch put ssl failed: %w", err)
		}
		if err = s.batchPutResource(ctx, ApisixResourceTypePluginMetadata, putConf.PluginMetadatas); err != nil {
			return fmt.Errorf("batch put plugin metadata failed: %w", err)
		}
		if err = s.batchPutResource(ctx, ApisixResourceTypeServices, putConf.Services); err != nil {
			return fmt.Errorf("batch put services failed: %w", err)
		}

		// sleep putInterVal to avoid resource data inconsistency
		time.Sleep(s.putInterval)

		if err = s.batchPutResource(ctx, ApisixResourceTypeRoutes, putConf.Routes); err != nil {
			return fmt.Errorf("batch put routes failed: %w", err)
		}

		if err = s.batchPutResource(ctx, ApisixResourceTypeStreamRoutes, putConf.StreamRoutes); err != nil {
			return fmt.Errorf("batch put stream routes failed: %w", err)
		}

		if len(putConf.Routes)+len(putConf.StreamRoutes)+len(putConf.Services)+
			len(putConf.PluginMetadatas)+len(putConf.SSLs) > 0 {
			s.logger.Infof(
				"put gateway[key=%s] conf count:[route:%d,stream_route:%d,serivce:%d,plugin_metadata:%d,ssl:%d]",
				stageKey,
				len(putConf.Routes),
				len(putConf.StreamRoutes),
				len(putConf.Services),
				len(putConf.PluginMetadatas),
				len(putConf.SSLs),
			)
		}
	}

	// delete resources
	if deleteConf != nil {
		// NOTE: 删除的顺序和创建的顺序相反, 错误的顺序会导致apisix的异常
		if err = s.batchDeleteResource(ctx, ApisixResourceTypeStreamRoutes, deleteConf.StreamRoutes); err != nil {
			return fmt.Errorf("batch delete stream routes failed: %w", err)
		}
		if err = s.batchDeleteResource(ctx, ApisixResourceTypeRoutes, deleteConf.Routes); err != nil {
			return fmt.Errorf("batch delete routes failed: %w", err)
		}
		if err = s.batchDeleteResource(ctx, ApisixResourceTypeServices, deleteConf.Services); err != nil {
			return fmt.Errorf("batch delete services failed: %w", err)
		}
		if err = s.batchDeleteResource(ctx, ApisixResourceTypePluginMetadata, deleteConf.PluginMetadatas); err != nil {
			return fmt.Errorf("batch delete plugin metadata failed: %w", err)
		}
		if err = s.batchDeleteResource(ctx, ApisixResourceTypeSSL, deleteConf.SSLs); err != nil {
			return fmt.Errorf("batch delete ssl failed: %w", err)
		}
		if len(deleteConf.Routes)+len(deleteConf.StreamRoutes)+len(deleteConf.Services)+
			len(deleteConf.PluginMetadatas)+len(deleteConf.SSLs) > 0 {
			s.logger.Infof(
				"del gateway[key=%s] conf count:[route:%d,stream_route:%d,serivce:%d,plugin_metadata:%d,ssl:%d]",
				stageKey,
				len(deleteConf.Routes),
				len(deleteConf.StreamRoutes),
				len(deleteConf.Services),
				len(deleteConf.PluginMetadatas),
				len(deleteConf.SSLs),
			)
		}
	}

	if deleteConf == nil && putConf == nil {
		s.logger.Infof("%s has no change", stageKey)
	}

	return nil
}

func (s *EtcdConfigStore) batchPutResource(ctx context.Context, resourceType string, resources interface{}) error {
	resourceStore := s.stores[resourceType]

	resourceIter := reflect.ValueOf(resources).MapRange()
	for resourceIter.Next() {
		// set create time from cache resource
		st := time.Now()

		key := resourceIter.Key().Interface().(string)
		resource := resourceIter.Value().Interface().(apisix.ApisixResource)
		oldSt := resourceStore.getResourceCreateTime(resource.GetID())
		if oldSt != 0 {
			resource.SetCreateTime(oldSt)
		} else {
			resource.SetCreateTime(st.Unix())
		}

		resource.SetUpdateTime(st.Unix())

		bytes, err := json.Marshal(resource)
		if err != nil {
			s.logger.Error(
				"Marshal resource failed",
				"err",
				err,
				"resourceType",
				resourceType,
				"resourceID",
				resource.GetID(),
			)
			return fmt.Errorf("marshal resource failed: %w", err)
		}

		s.logger.Debugw("Put resource to etcd", "resourceType", resourceType, "resourceID", resource.GetID())

		_, err = s.client.Put(ctx, resourceStore.prefix+key, string(bytes))

		synchronizer.ReportApisixEtcdMetric(resourceType, metric.ActionPut, st, err)

		if err != nil {
			s.logger.Errorw(
				"Put resource failed",
				"err",
				err,
				"resourceType",
				resourceType,
				"resourceID",
				resource.GetID(),
			)
			return fmt.Errorf("put resource failed: %w", err)
		}
	}
	return nil
}

func (s *EtcdConfigStore) batchDeleteResource(ctx context.Context, resourceType string, resources interface{}) error {
	resourceStore := s.stores[resourceType]
	resourceMap := reflect.ValueOf(resources).MapRange()
	for resourceMap.Next() {
		st := time.Now()

		key := resourceMap.Key().Interface().(string)
		resource := resourceMap.Value().Interface().(apisix.ApisixResource)

		s.logger.Debugw("Delete resource from etcd", "resourceType", resourceType, "resourceID", resource.GetID())

		_, err := s.client.Delete(ctx, resourceStore.prefix+key)

		synchronizer.ReportApisixEtcdMetric(resourceType, metric.ActionDelete, st, err)

		if err != nil {
			s.logger.Errorw(
				"Delete resource failed",
				"err",
				err,
				"resourceType",
				resourceType,
				"resourceID",
				resource.GetID(),
			)
			return fmt.Errorf("delete resource failed: %w", err)
		}
	}

	return nil
}
