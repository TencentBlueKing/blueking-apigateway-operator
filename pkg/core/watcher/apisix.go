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

// Package watcher ...
package watcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

type ApisixWatcher struct {
	client *clientv3.Client
	Prefix string // example: /apisix/routes

	mux       sync.RWMutex
	resources map[string]entity.ApisixResource // resource id -> resource

	currentRevision int64

	logger *zap.SugaredLogger
}

// NewApisixResourceWatcher creates a new ApisixWatcher instance
func NewApisixResourceWatcher(client *clientv3.Client, prefix string) (*ApisixWatcher, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	apisixWatcher := &ApisixWatcher{
		client: client,
		Prefix: prefix,
		logger: logging.GetLogger().Named("etcd-resource-store"),
	}

	apisixWatcher.logger.Infow("Create etcd resource store", "Prefix", prefix)

	err := apisixWatcher.fullSync(context.Background())
	if err != nil {
		apisixWatcher.logger.Error(err, "full sync failed")
		return nil, fmt.Errorf("init local resource store Prefix: %s error: %w", apisixWatcher.Prefix, err)
	}
	go apisixWatcher.incrSync()

	return apisixWatcher, nil
}

func (e *ApisixWatcher) fullSync(ctx context.Context) error {
	e.logger.Infow("etcdLocalResourceStore start full sync", "Prefix", e.Prefix)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ret, err := e.client.Get(ctx, e.Prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error(err, "List resource from etcd failed")
		return err
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	e.resources = make(map[string]entity.ApisixResource)

	for i := range ret.Kvs {
		resource, err := e.addResource(ret.Kvs[i].Key, ret.Kvs[i].Value)
		if err != nil {
			e.logger.Error(err, "Parse resource from etcd failed")
			continue
		}

		if resource == nil {
			continue
		}

		e.logger.Debugw("store resource", "key", string(ret.Kvs[i].Key), "resourceID",
			resource.GetID())
		e.resources[resource.GetID()] = resource
	}

	e.currentRevision = ret.Header.Revision
	return nil
}

func (e *ApisixWatcher) addResource(key, value []byte) (resource entity.ApisixResource, err error) {
	if len(e.Prefix) == len(key) {
		return nil, nil
	}
	if string(value) == constant.SkippedValueEtcdInitDir ||
		string(value) == constant.SkippedValueEtcdEmptyObject {
		return nil, nil
	}

	parts := strings.Split(strings.Trim(e.Prefix, "/"), "/")
	if len(parts) == 0 {
		e.logger.Error("Invalid Prefix key", e.Prefix)
		return nil, fmt.Errorf("invalid Prefix key: %s", e.Prefix)
	}
	resourceType := parts[len(parts)-1]
	switch resourceType {
	case constant.ApisixResourceTypeRoutes:
		resource = &entity.Route{}
	case constant.ApisixResourceTypeServices:
		resource = &entity.Service{}
	case constant.ApisixResourceTypeSSL:
		resource = &entity.SSL{}
	case constant.ApisixResourceTypePluginMetadata:
		var metadata entity.ResourceMetadata
		err = json.Unmarshal(value, &metadata)
		if err != nil {
			e.logger.Error("Unmarshal resource from etcd failed")
			return nil, fmt.Errorf("unmarshal resource from etcd failed: %w", err)
		}
		resource = &entity.PluginMetadata{
			ResourceMetadata: metadata,
			PluginMetadataConf: entity.PluginMetadataConf{
				metadata.GetID(): value,
			},
		}
	default:
		e.logger.Errorw("Unknown resource type", "resourceType", resourceType)
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	if resourceType != constant.ApisixResourceTypePluginMetadata {
		err = json.Unmarshal(value, resource)
		if err != nil {
			e.logger.Error("Unmarshal resource from etcd failed")
			return nil, fmt.Errorf("unmarshal resource from etcd failed: %w", err)
		}
	}
	return resource, nil
}

// nolint: staticcheck
func (e *ApisixWatcher) incrSync() {
	c, cancel := context.WithCancel(context.TODO())
	defer cancel()
	var ch clientv3.WatchChan
	needCreateChan := true
	for {
		if needCreateChan {
			ch = e.client.Watch(
				clientv3.WithRequireLeader(c),
				e.Prefix,
				clientv3.WithPrefix(),
				clientv3.WithPrevKV(),
				clientv3.WithRev(e.currentRevision),
			)
			needCreateChan = false
		}

		select {
		case event, ok := <-ch:
			if !ok || event.Err() != nil {
				e.logger.Error(event.Err(), "Watch event failed")

				time.Sleep(constant.SyncSleepSeconds)

				switch err := event.Err(); {
				case errors.Is(err, v3rpc.ErrCompacted), errors.Is(err, v3rpc.ErrFutureRev):
					err := e.fullSync(c)
					if err != nil {
						time.Sleep(constant.SyncSleepSeconds)
						continue
					}
				}
				// reset channel
				needCreateChan = true
				cancel()
				c, cancel = context.WithCancel(context.TODO())
				break
			}
			// handler event
			for _, evt := range event.Events {
				err := e.handlerEvent(evt)
				if err != nil {
					e.logger.Errorf("Handle event failed:%v", err)
					continue
				}
			}
			e.currentRevision = event.Header.Revision
		}
	}
}

func (e *ApisixWatcher) handlerEvent(event *clientv3.Event) error {
	switch event.Type {
	case clientv3.EventTypePut:
		resource, err := e.addResource(event.Kv.Key, event.Kv.Value)
		if err != nil {
			e.logger.Error(err, "Parse resource from etcd failed")
			return err
		}

		if resource == nil {
			return errors.New("resource is nil")
		}

		e.logger.Debugw(
			"Put resource",
			"key",
			string(event.Kv.Key),
			"resourceID",
			resource.GetID(),
		)
		e.mux.Lock()
		e.resources[resource.GetID()] = resource
		e.mux.Unlock()
	case clientv3.EventTypeDelete:
		resource, err := e.addResource(event.PrevKv.Key, event.PrevKv.Value)
		if err != nil {
			e.logger.Error(err, "Parse resource from etcd failed")
			return err
		}

		if resource == nil {
			return errors.New("resource is nil")
		}

		e.logger.Debugw(
			"Delete resource",
			"key",
			string(event.Kv.Key),
			"resourceID",
			resource.GetID(),
		)
		e.mux.Lock()
		delete(e.resources, resource.GetID())
		e.mux.Unlock()
	}
	return nil
}

// GetStageResources returns all resources for a specific stage
func (e *ApisixWatcher) GetStageResources(stageName string) map[string]entity.ApisixResource {
	e.mux.RLock()
	defer e.mux.RUnlock()
	resources := make(map[string]entity.ApisixResource)
	for key, resource := range e.resources {
		if resource.GetReleaseInfo() == nil {
			continue
		}
		stageKey := resource.GetReleaseInfo().GetStageKey()
		if stageKey == stageName {
			resources[key] = resource
		}
	}
	return resources
}

// GetAllResources returns all resources from the watcher
func (e *ApisixWatcher) GetAllResources() map[string]entity.ApisixResource {
	e.mux.RLock()
	defer e.mux.RUnlock()

	resources := make(map[string]entity.ApisixResource)
	for key, resource := range e.resources {
		resources[key] = resource
	}

	return resources
}
