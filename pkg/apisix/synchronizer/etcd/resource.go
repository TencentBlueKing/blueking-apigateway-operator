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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

const (
	// skippedValueEtcdInitDir indicates the init_dir
	// etcd event will be skipped.
	skippedValueEtcdInitDir = "init_dir"

	// skippedValueEtcdEmptyObject indicates the data with an
	// empty JSON value {}, which may be set by APISIX,
	// should be also skipped.
	//
	// Important: at present, {} is considered as invalid,
	// but may be changed in the future.
	skippedValueEtcdEmptyObject = "{}"

	ApisixResourceTypeRoutes         = "routes"
	ApisixResourceTypeStreamRoutes   = "stream_routes"
	ApisixResourceTypeServices       = "services"
	ApisixResourceTypeSSL            = "ssls"
	ApisixResourceTypePluginMetadata = "plugin_metadata"

	syncSleepSeconds = 5 * time.Second
)

type resourceStore struct {
	client *clientv3.Client
	prefix string // example: /apisix/routes

	mux       sync.RWMutex
	resources map[string]apisix.ApisixResource // resource id -> resource

	currentRevision int64

	logger *zap.SugaredLogger
}

func newResourceStore(client *clientv3.Client, prefix string) (*resourceStore, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	store := &resourceStore{
		client: client,
		prefix: prefix,
		logger: logging.GetLogger().Named("etcd-resource-store"),
	}

	store.logger.Infow("Create etcd resource store", "prefix", prefix)

	err := store.fullSync(context.Background())
	if err != nil {
		store.logger.Error(err, "full sync failed")
		return nil, fmt.Errorf("init local resource store prefix: %s error: %w", store.prefix, err)
	}
	go store.incrSync()

	return store, nil
}

func (e *resourceStore) fullSync(ctx context.Context) error {
	e.logger.Infow("etcdLocalResourceStore start full sync", "prefix", e.prefix)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ret, err := e.client.Get(ctx, e.prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error(err, "List resource from etcd failed")
		return err
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	e.resources = make(map[string]apisix.ApisixResource)

	for i := range ret.Kvs {
		resource, err := e.parseResource(ret.Kvs[i].Key, ret.Kvs[i].Value)
		if err != nil {
			e.logger.Error(err, "Parse resource from etcd failed")
			continue
		}

		if resource == nil {
			continue
		}

		e.logger.Debugw("store resource", "key", string(ret.Kvs[i].Key), "resourceID", resource.GetID())
		e.resources[resource.GetID()] = resource
	}

	e.currentRevision = ret.Header.Revision
	return nil
}

func (e *resourceStore) parseResource(key, value []byte) (resource apisix.ApisixResource, err error) {
	if len(e.prefix) == len(key) {
		return nil, nil
	}
	if string(value) == skippedValueEtcdInitDir ||
		string(value) == skippedValueEtcdEmptyObject {
		return nil, nil
	}

	parts := strings.Split(strings.Trim(e.prefix, "/"), "/")
	if len(parts) == 0 {
		e.logger.Error("Invalid prefix key", e.prefix)
		return nil, fmt.Errorf("invalid prefix key: %s", e.prefix)
	}
	resourceType := parts[len(parts)-1]

	switch resourceType {
	case ApisixResourceTypeRoutes:
		resource = &apisix.Route{}
	case ApisixResourceTypeStreamRoutes:
		resource = &apisix.StreamRoute{}
	case ApisixResourceTypeServices:
		resource = &apisix.Service{}
	case ApisixResourceTypeSSL:
		resource = &apisix.SSL{}
	case ApisixResourceTypePluginMetadata:
		resource = &apisix.PluginMetadata{}
	default:
		e.logger.Errorw("Unknown resource type", "resourceType", resourceType)
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	err = json.Unmarshal(value, resource)
	if err != nil {
		e.logger.Error("Unmarshal resource from etcd failed")
		return nil, fmt.Errorf("unmarshal resource from etcd failed: %w", err)
	}

	// remove resource desc
	resource.ClearUnusedFields()
	return resource, nil
}

//nolint:gosimple
func (e *resourceStore) incrSync() {
	c, cancel := context.WithCancel(context.TODO())
	defer cancel()
	var ch clientv3.WatchChan
	needCreateChan := true
	for {
		if needCreateChan {
			ch = e.client.Watch(
				clientv3.WithRequireLeader(c),
				e.prefix,
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

				time.Sleep(syncSleepSeconds)

				switch err := event.Err(); {
				case errors.Is(err, v3rpc.ErrCompacted), errors.Is(err, v3rpc.ErrFutureRev):
					err := e.fullSync(c)
					if err != nil {
						time.Sleep(syncSleepSeconds)
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

func (e *resourceStore) handlerEvent(event *clientv3.Event) error {
	switch event.Type {
	case clientv3.EventTypePut:
		resource, err := e.parseResource(event.Kv.Key, event.Kv.Value)
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
		resource, err := e.parseResource(event.PrevKv.Key, event.PrevKv.Value)
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

func (e *resourceStore) getResourceCreateTime(resourceID string) int64 {
	e.mux.RLock()
	defer e.mux.RUnlock()

	resource, ok := e.resources[resourceID]
	if !ok {
		return 0
	}

	return resource.GetCreateTime()
}

func (e *resourceStore) getStageResources(stageName string) map[string]apisix.ApisixResource {
	e.mux.RLock()
	defer e.mux.RUnlock()

	resources := make(map[string]apisix.ApisixResource)
	for key, resource := range e.resources {
		if resource.GetStageFromLabel() == stageName {
			resources[key] = resource
		}
	}
	return resources
}

func (e *resourceStore) getAllResources() map[string]apisix.ApisixResource {
	e.mux.RLock()
	defer e.mux.RUnlock()

	resources := make(map[string]apisix.ApisixResource)
	for key, resource := range e.resources {
		resources[key] = resource
	}

	return resources
}
