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

// Package watcher ...
package watcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
)

// APIGEtcdWWatcher implements the Register interface using etcd as the main storage.
type APIGEtcdWWatcher struct {
	etcdClient *clientv3.Client

	logger *zap.SugaredLogger

	keyPrefix string

	currentRevision int64
}

// NewEtcdResourceRegistry creates a APIGEtcdWWatcher object
func NewEtcdResourceRegistry(etcdClient *clientv3.Client, keyPrefix string) *APIGEtcdWWatcher {
	registry := &APIGEtcdWWatcher{
		etcdClient: etcdClient,
	}
	registry.logger = logging.GetLogger().Named("registry")

	registry.keyPrefix = keyPrefix
	return registry
}

// Watch creates and returns a channel that produces update events of resources.
func (r *APIGEtcdWWatcher) Watch(ctx context.Context) <-chan *entity.ResourceMetadata {
	watchCtx, cancel := context.WithCancel(ctx)
	retCh := make(chan *entity.ResourceMetadata)
	var etcdWatchCh clientv3.WatchChan
	needCreateChan := true
	go func() {
		defer func() {
			r.currentRevision = 0
			close(retCh)
		}()

		for {
			if needCreateChan {
				etcdWatchCh = r.etcdClient.Watch(
					clientv3.WithRequireLeader(watchCtx),
					strings.TrimSuffix(r.keyPrefix, "/")+"/",
					clientv3.WithPrefix(),
					clientv3.WithPrevKV(),
					clientv3.WithRev(r.currentRevision),
				)
				needCreateChan = false
			}
			select {
			case event, ok := <-etcdWatchCh:
				// reset watch channel if get error
				if !ok {
					r.logger.Error(nil, "Watch etcd registry failed: channel break, will recover from cached revision",
						"revision",
						r.currentRevision,
					)
					time.Sleep(time.Second * 5)
					needCreateChan = true
					cancel()
					watchCtx, cancel = context.WithCancel(ctx)
					break
				}

				r.logger.Debugw("etcd event trigger", "event", event)
				err := event.Err()
				if err != nil {
					switch {
					case errors.Is(err, v3rpc.ErrCompacted), errors.Is(err, v3rpc.ErrFutureRev):
						r.logger.Error(event.Err(),
							"Watch etcd registry failed unrecoverable, need full sync to recover",
						)
						return
					default:
						r.logger.Error(event.Err(),
							"Watch etcd registry failed: other error, will recover from cached revision",
							"revision", r.currentRevision,
						)
						time.Sleep(time.Second * 5)
						needCreateChan = true
						cancel()
						watchCtx, cancel = context.WithCancel(ctx)
					}
					// break select
					break
				}
				for _, evt := range event.Events {
					metadata, handleErr := r.handleEvent(evt)
					if handleErr != nil {
						r.logger.Errorf("handle etcd event failed:%v", handleErr)
						continue
					}
					retCh <- metadata
					r.currentRevision = event.Header.Revision
				}

			case <-ctx.Done():
				r.logger.Infow("stop etcd watch loop canceled by context")
				return
			}
		}
	}()
	return retCh
}

// handle event
func (r *APIGEtcdWWatcher) handleEvent(event *clientv3.Event) (*entity.ResourceMetadata, error) {
	switch event.Type {
	case clientv3.EventTypePut:
		r.logger.Debugw(
			"Etcd Put events triggeres",
			"action",
			event.Type,
			"key",
			string(event.Kv.Key),
			"value",
			string(event.Kv.Value),
		)
		// trace
		metadata, err := r.extractResourceMetadata(string(event.Kv.Key), event.Kv.Value)
		eventCtx, span := trace.StartTrace(metadata.Ctx, "registry.EventPut")
		defer span.End()
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		span.SetAttributes(
			attribute.String("resource.name", metadata.Name),
			attribute.String("stage", metadata.GetStageName()),
			attribute.String("gateway", metadata.GetGatewayName()),
			attribute.String("resource.kind", metadata.Kind.String()),
		)
		metadata.Ctx = eventCtx
		return &metadata, nil
	case clientv3.EventTypeDelete:
		r.logger.Debugw(
			"Etcd Delete events triggeres",
			"action",
			event.Type,
			"key",
			string(event.PrevKv.Key),
			"value",
			string(event.PrevKv.Value),
		)
		metadata, err := r.extractResourceMetadata(string(event.PrevKv.Key), event.PrevKv.Value)
		// trace
		eventCtx, span := trace.StartTrace(metadata.Ctx, "registry.EventDelete")
		defer span.End()
		span.SetAttributes(
			attribute.String("resource.name", metadata.Name),
			attribute.String("stage", metadata.GetStageName()),
			attribute.String("gateway", metadata.GetGatewayName()),
			attribute.String("resource.kind", metadata.Kind.String()),
		)
		if err != nil {
			r.logger.Infow(
				"deleted resource key is incorrect, skip it",
				"key",
				string(event.PrevKv.Key),
				"value",
				string(event.PrevKv.Value),
				"err",
				err,
			)
			span.RecordError(err)
			return nil, err
		}
		metadata.Ctx = eventCtx
		return &metadata, nil
	}
	return nil, fmt.Errorf("err unknown event type: %s", event.Type)
}

func (r *APIGEtcdWWatcher) extractResourceMetadata(key string, value []byte) (entity.ResourceMetadata, error) {
	// /{self.prefix}/{self.api_version}/gateway/{gateway_name}/{stage_name}/route/bk-default.default.-1
	ret := entity.ResourceMetadata{}
	err := json.Unmarshal(value, &ret)
	if err != nil {
		r.logger.Error(err, "unmarshal etcd value failed", "value", string(value))
		return ret, err
	}
	if len(key) == 0 {
		r.logger.Error(nil, "empty key", "key", key)
		return ret, eris.Errorf("empty key")
	}
	// remove leading /
	matches := strings.Split(key[1:], "/")
	if matches == nil {
		r.logger.Error(nil, "regex match failed, not found", "key", key)
		return ret, eris.Errorf("regex match failed, not found")
	}
	if len(matches) < 7 {
		r.logger.Error(nil, "Etcd key segment by slash should larger or equal to 5", "key", key)
		return ret, eris.Errorf("Etcd key segment by slash should larger or equal to 5")
	}

	ret.APIVersion = matches[len(matches)-6]
	ret.Kind = constant.APISIXResource(matches[len(matches)-2])
	ret.Name = matches[len(matches)-1]
	ret.Ctx = context.Background()
	r.logger.Debugw("Extract resource info from etcdkey", "key", key, "resourceInfo", ret)
	return ret, nil
}
