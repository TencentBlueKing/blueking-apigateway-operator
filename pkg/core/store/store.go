/*
 *  TencentBlueKing is pleased to support the open source community by making
 *  蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 *  Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 *  Licensed under the MIT License (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 *      http://opensource.org/licenses/MIT
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 *  either express or implied. See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 *   We undertake not to change the open source license (MIT license) applicable
 *   to the current version of the project delivered to anyone in the future.
 */

// Package store ...
package store

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

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/differ"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/watcher"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

var apisixResourceTypes = []string{
	constant.ApisixResourceTypeRoutes,
	constant.ApisixResourceTypeServices,
	constant.ApisixResourceTypeSSL,
	constant.ApisixResourceTypePluginMetadata,
}

// ApisixEtcdConfigStore ...
type ApisixEtcdConfigStore struct {
	client *clientv3.Client
	prefix string

	watcher map[string]*watcher.ApisixWatcher
	differ  *differ.ConfigDiffer

	logger *zap.SugaredLogger

	putInterval time.Duration

	delInterval time.Duration

	lock *sync.RWMutex
}

// NewApiEtcdConfigStore ...
func NewApiEtcdConfigStore(client *clientv3.Client, prefix string,
	putInterval time.Duration, delInterval time.Duration) (*ApisixEtcdConfigStore, error) {
	s := &ApisixEtcdConfigStore{
		client:      client,
		prefix:      strings.TrimRight(prefix, "/"),
		watcher:     make(map[string]*watcher.ApisixWatcher, 4),
		differ:      differ.NewConfigDiffer(),
		logger:      logging.GetLogger().Named("etcd-config-store"),
		putInterval: putInterval,
		delInterval: delInterval,
		lock:        &sync.RWMutex{},
	}
	s.Init()

	s.logger.Infow("Create etcd config store", "prefix", prefix)

	if len(s.watcher) != len(apisixResourceTypes) {
		s.logger.Error("Create etcd config store failed")
		return nil, fmt.Errorf("create etcd config store failed")
	}

	return s, nil
}

// Init initializes the etcd config store
func (s *ApisixEtcdConfigStore) Init() {
	wg := &sync.WaitGroup{}
	for _, resourceType := range apisixResourceTypes {
		wg.Add(1)

		// 避免闭包导致变量覆盖问题
		tempResourceType := resourceType
		utils.GoroutineWithRecovery(context.Background(), func() {
			defer wg.Done()
			resourceStore, err := watcher.NewApisixResourceWatcher(s.client, s.prefix+"/"+tempResourceType+"/")
			if err != nil {
				s.logger.Errorw("Create resource store failed", "resourceType", tempResourceType)
				return
			}
			s.lock.Lock()
			defer s.lock.Unlock()
			s.watcher[tempResourceType] = resourceStore
		})
	}
	wg.Wait()
}

// Get get a staged apisix configuration
func (s *ApisixEtcdConfigStore) Get(stageName string) *entity.ApisixStageResource {
	ret := entity.NewEmptyApisixConfiguration()
	routes := s.watcher[constant.ApisixResourceTypeRoutes].GetStageResources(stageName)
	for key, val := range routes {
		ret.Routes[key] = val.(*entity.Route)
	}
	services := s.watcher[constant.ApisixResourceTypeServices].GetStageResources(stageName)
	for key, val := range services {
		ret.Services[key] = val.(*entity.Service)
	}
	ssls := s.watcher[constant.ApisixResourceTypeSSL].GetStageResources(stageName)
	for key, val := range ssls {
		ret.SSLs[key] = val.(*entity.SSL)
	}
	pms := s.watcher[constant.ApisixResourceTypePluginMetadata].GetStageResources(stageName)
	for key, val := range pms {
		ret.PluginMetadatas[key] = val.(*entity.PluginMetadata)
	}
	return ret
}

// GetAll get staged apisix configuration map
func (s *ApisixEtcdConfigStore) GetAll() map[string]*entity.ApisixStageResource {
	configMap := make(map[string]*entity.ApisixStageResource)
	routeMap := s.watcher[constant.ApisixResourceTypeRoutes].GetAllResources()
	for key, route := range routeMap {
		stageName := route.GetStageName()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = entity.NewEmptyApisixConfiguration()
		}
		configMap[stageName].Routes[key] = route.(*entity.Route)
	}

	serviceMap := s.watcher[constant.ApisixResourceTypeServices].GetAllResources()
	for key, service := range serviceMap {
		stageName := service.GetStageName()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = entity.NewEmptyApisixConfiguration()
		}
		configMap[stageName].Services[key] = service.(*entity.Service)
	}

	sslMap := s.watcher[constant.ApisixResourceTypeSSL].GetAllResources()
	for key, ssl := range sslMap {
		stageName := ssl.GetStageName()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = entity.NewEmptyApisixConfiguration()
		}
		configMap[stageName].SSLs[key] = ssl.(*entity.SSL)
	}

	pmMap := s.watcher[constant.ApisixResourceTypePluginMetadata].GetAllResources()
	for key, pm := range pmMap {
		stageName := pm.GetStageName()
		if _, ok := configMap[stageName]; !ok {
			configMap[stageName] = entity.NewEmptyApisixConfiguration()
		}
		configMap[stageName].PluginMetadatas[key] = pm.(*entity.PluginMetadata)
	}

	return configMap
}

// Alter ...
func (s *ApisixEtcdConfigStore) Alter(
	ctx context.Context,
	stageName string,
	config *entity.ApisixStageResource,
) error {
	st := time.Now()
	err := s.alterByStage(ctx, stageName, config)

	// metric
	metric.ReportStageConfigAlterMetric(stageName, config, st, err)

	if err != nil {
		s.logger.Errorw("Alter by stage failed", "err", err, "stage", stageName)
		return err
	}

	return nil
}

func (s *ApisixEtcdConfigStore) alterByStage(
	ctx context.Context, stageKey string, conf *entity.ApisixStageResource,
) (err error) {
	// get cached config
	oldConf := s.Get(stageKey)

	// diff config
	putConf, deleteConf := s.differ.Diff(oldConf, conf)
	// put resources
	if putConf != nil {
		if err = s.batchPutResource(ctx, constant.ApisixResourceTypeSSL, putConf.SSLs); err != nil {
			return fmt.Errorf("batch put ssl failed: %w", err)
		}
		if err = s.batchPutResource(ctx, constant.ApisixResourceTypePluginMetadata, putConf.PluginMetadatas); err != nil {
			return fmt.Errorf("batch put plugin metadata failed: %w", err)
		}
		if err = s.batchPutResource(ctx, constant.ApisixResourceTypeServices, putConf.Services); err != nil {
			return fmt.Errorf("batch put services failed: %w", err)
		}

		// sleep putInterVal to avoid resource data inconsistency
		time.Sleep(s.putInterval)

		if err = s.batchPutResource(ctx, constant.ApisixResourceTypeRoutes, putConf.Routes); err != nil {
			return fmt.Errorf("batch put routes failed: %w", err)
		}

		if len(putConf.Routes)+len(putConf.Services)+
			len(putConf.PluginMetadatas)+len(putConf.SSLs) > 0 {
			s.logger.Infof(
				"put gateway[key=%s] conf count:[route:%d,serivce:%d,plugin_metadata:%d,ssl:%d]",
				stageKey,
				len(putConf.Routes),
				len(putConf.Services),
				len(putConf.PluginMetadatas),
				len(putConf.SSLs),
			)
		}
	}

	// delete resources
	if deleteConf != nil {
		if err = s.batchDeleteResource(ctx, constant.ApisixResourceTypeRoutes, deleteConf.Routes); err != nil {
			return fmt.Errorf("batch delete routes failed: %w", err)
		}

		if err = s.batchDeleteResource(ctx, constant.ApisixResourceTypePluginMetadata, deleteConf.PluginMetadatas); err != nil {
			return fmt.Errorf("batch delete plugin metadata failed: %w", err)
		}
		if err = s.batchDeleteResource(ctx, constant.ApisixResourceTypeSSL, deleteConf.SSLs); err != nil {
			return fmt.Errorf("batch delete ssl failed: %w", err)
		}

		if len(deleteConf.Services) > 0 {
			// sleep delInterval to avoid resource data inconsistency
			time.Sleep(s.delInterval)
			if err = s.batchDeleteResource(ctx, constant.ApisixResourceTypeServices, deleteConf.Services); err != nil {
				return fmt.Errorf("batch delete services failed: %w", err)
			}
		}
	}

	if deleteConf == nil && putConf == nil {
		s.logger.Infof("%s has no change", stageKey)
	}

	return nil
}

func (s *ApisixEtcdConfigStore) batchPutResource(ctx context.Context, resourceType string, resources interface{}) error {
	resourceStore := s.watcher[resourceType]

	resourceIter := reflect.ValueOf(resources).MapRange()
	for resourceIter.Next() {
		// set create time from cache resource
		st := time.Now()
		key := resourceIter.Key().Interface().(string)
		resource := resourceIter.Value().Interface().(entity.ApisixResource)
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

		_, err = s.client.Put(ctx, resourceStore.Prefix+key, string(bytes))

		metric.ReportApisixEtcdMetric(resourceType, metric.ActionPut, st, err)

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

func (s *ApisixEtcdConfigStore) batchDeleteResource(ctx context.Context, resourceType string, resources interface{}) error {
	resourceStore := s.watcher[resourceType]
	resourceMap := reflect.ValueOf(resources).MapRange()
	for resourceMap.Next() {
		st := time.Now()

		key := resourceMap.Key().Interface().(string)
		resource := resourceMap.Value().Interface().(entity.ApisixResource)

		s.logger.Debugw("Delete resource from etcd", "resourceType", resourceType, "resourceID", resource.GetID())

		_, err := s.client.Delete(ctx, resourceStore.Prefix+key)

		metric.ReportApisixEtcdMetric(resourceType, metric.ActionDelete, st, err)

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
