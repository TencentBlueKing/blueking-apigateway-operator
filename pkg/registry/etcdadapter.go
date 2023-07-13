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

package registry

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	"go.etcd.io/etcd/api/v3/mvccpb"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/internal/tracer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

// EtcdRegistryAdapter implements the Register interface using etcd as the main storage.
type EtcdRegistryAdapter struct {
	etcdClient *clientv3.Client

	logger *zap.SugaredLogger

	keyPrefix string

	currentRevision int64
}

// NewEtcdResourceRegistry creates a EtcdRegistryAdapter object
func NewEtcdResourceRegistry(etcdClient *clientv3.Client, keyPrefix string) *EtcdRegistryAdapter {
	registry := &EtcdRegistryAdapter{
		etcdClient: etcdClient,
	}
	registry.logger = logging.GetLogger().Named("registry")

	registry.keyPrefix = keyPrefix
	return registry
}

func (r *EtcdRegistryAdapter) startSpan(
	ctx context.Context,
	key ResourceKey,
	kind string,
) (context.Context, trace.Span) {
	return tracer.NewTracer("registry").Start(
		ctx,
		"etcdRegistry."+kind,
		trace.WithAttributes(
			attribute.String("res.name", key.ResourceName),
			attribute.String("stage", key.StageName),
			attribute.String("gateway", key.GatewayName),
		),
	)
}

// Get ...
func (r *EtcdRegistryAdapter) Get(ctx context.Context, key ResourceKey, obj client.Object) error {
	startedTime := time.Now()
	gvk, ok := v1beta1.GetGVK(obj)
	if !ok {
		return eris.Errorf("Get gvk from object failed, key: %+v", key)
	}

	// tracing
	var span trace.Span
	ctx, span = r.startSpan(ctx, key, gvk.Kind)
	defer span.End()

	etcdKey := fmt.Sprintf(
		"%s/%s/%s/%s/%s/%s",
		r.keyPrefix,
		key.GatewayName,
		key.StageName,
		gvk.Version,
		gvk.Kind,
		key.ResourceName,
	)

	// 1. get value from etcd
	resp, err := r.etcdClient.Get(ctx, etcdKey)
	if err != nil {
		metric.ReportRegistryAction(gvk.Kind, metric.ActionGet, metric.ResultFail, startedTime)
		return err
	}

	if resp.Count == 0 {
		metric.ReportRegistryAction(gvk.Kind, metric.ActionGet, metric.ResultFail, startedTime)
		return k8serrors.NewNotFound(schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}, key.ResourceName)
	}

	// 2. convert yaml formated spec content into map
	err = r.yamlUnmarshal(gvk, resp.Kvs[0], obj)
	if err != nil {
		metric.ReportRegistryAction(gvk.Kind, metric.ActionGet, metric.ResultFail, startedTime)
		return err
	}
	obj.SetName(key.ResourceName)

	metric.ReportRegistryAction(gvk.Kind, metric.ActionGet, metric.ResultSuccess, startedTime)

	return nil
}

// ListStages ...
func (r *EtcdRegistryAdapter) ListStages(ctx context.Context) ([]StageInfo, error) {
	startedTime := time.Now()
	// 1. query from etcd
	resp, err := r.etcdClient.Get(
		ctx,
		strings.TrimSuffix(r.keyPrefix, "/")+"/",
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
	)
	if err != nil {
		metric.ReportRegistryAction(v1beta1.BkGatewayStageTypeName, metric.ActionList, metric.ResultFail, startedTime)
		return nil, err
	}

	// 2. convert
	stageList := make([]StageInfo, 0)
	stageMap := make(map[StageInfo]struct{})
	for _, kv := range resp.Kvs {
		rm, err := r.extractResourceMetadata(string(kv.Key))
		if err != nil {
			r.logger.Infow("resource key is incorrect, skip it", "key", string(kv.Key), "err", err)
			continue
		}
		stageMap[rm.StageInfo] = struct{}{}
	}
	for stage := range stageMap {
		stageList = append(stageList, stage)
	}

	metric.ReportRegistryAction(v1beta1.BkGatewayStageTypeName, metric.ActionList, metric.ResultSuccess, startedTime)

	return stageList, nil
}

// List ...
func (r *EtcdRegistryAdapter) List(ctx context.Context, key ResourceKey, obj client.ObjectList) error {
	startedTime := time.Now()
	if key.GatewayName == "" || key.StageName == "" {
		return eris.Errorf("GatewayName and stage must be specified when list etcd resources")
	}

	// type of *BkGatewaResourceList
	objListType := reflect.TypeOf(obj)
	if objListType.Kind() != reflect.Ptr {
		return eris.Errorf("Input obj should be pointer to object list")
	}
	// value of BkGatewaResourceList
	objListValue := reflect.ValueOf(obj).Elem()
	// value of BkGatewaResourceList.Items ([]BkGatewayResource)
	objListItemsValue := objListValue.FieldByName("Items")
	// type of []BkGatewayResource
	objListItemsType := objListItemsValue.Type()
	// type of BkGatewayResource
	objTyp := objListItemsType.Elem()

	gvk, ok := v1beta1.GetGVK(obj)
	if !ok {
		metric.ReportRegistryAction(gvk.Kind, metric.ActionList, metric.ResultFail, startedTime)
		return eris.Errorf("Get gvk from object failed, key: %+v", key)
	}

	var span trace.Span
	ctx, span = r.startSpan(ctx, key, gvk.Kind)
	defer span.End()

	etcdKey := fmt.Sprintf("%s/%s/%s/%s/%s/", r.keyPrefix, key.GatewayName, key.StageName, gvk.Version, gvk.Kind)
	if key.ResourceName != "" {
		etcdKey += key.ResourceName
	}
	resp, err := r.etcdClient.Get(ctx, etcdKey, clientv3.WithPrefix())
	if err != nil {
		metric.ReportRegistryAction(gvk.Kind, metric.ActionList, metric.ResultFail, startedTime)
		return err
	}
	if resp.Count > math.MaxInt32 {
		r.logger.Error(
			nil,
			"etcd resource count is larger than MaxInt32, error may occurred in 32 bit CPU",
			"count",
			resp.Count,
			"key",
			key,
		)
	}

	valueList := reflect.MakeSlice(objListItemsType, int(resp.Count), int(resp.Count))
	cnt := 0
	for _, kv := range resp.Kvs {
		resMeta, err := r.extractResourceMetadata(string(kv.Key))
		if err != nil {
			r.logger.Infow("key is not resource key, maybe leader election keys, skip it", "key", string(kv.Key))
			continue
		}
		itemPtrValue := reflect.NewAt(objTyp, unsafe.Pointer(valueList.Index(cnt).UnsafeAddr()))
		itemObj, ok := itemPtrValue.Interface().(client.Object)

		if !ok {
			metric.ReportRegistryAction(gvk.Kind, metric.ActionList, metric.ResultFail, startedTime)
			r.logger.Error(
				nil,
				"Cast objlist item into client.Object failed",
				"key",
				key,
				"objListType",
				objListType,
				"itemType",
				objTyp,
				"inlistItemType",
				itemPtrValue.Type(),
			)
			return eris.Errorf("Cast objlist item into client.Object failed")
		}

		err = r.yamlUnmarshal(gvk, kv, itemObj)
		if err != nil {
			r.logger.Error(err, "unmarshal etcd resource to obj failed", "kv", kv, "key", key)
			continue
		}
		itemObj.SetName(resMeta.Name)
		cnt++
	}
	objListItemsValue.Set(valueList.Slice(0, cnt))

	metric.ReportRegistryAction(gvk.Kind, metric.ActionList, metric.ResultSuccess, startedTime)

	return nil
}

// Watch ...
func (r *EtcdRegistryAdapter) Watch(ctx context.Context) <-chan *ResourceMetadata {
	etcdWatchCh := r.etcdClient.Watch(
		ctx,
		strings.TrimSuffix(r.keyPrefix, "/")+"/",
		clientv3.WithPrefix(),
		clientv3.WithPrevKV(),
		clientv3.WithRev(r.currentRevision),
	)
	retCh := make(chan *ResourceMetadata)
	go func() {
		defer func() {
			r.currentRevision = 0
			close(retCh)
		}()

		for {
			select {
			case event, ok := <-etcdWatchCh:

				// reset watch channel if get error
				if !ok {
					r.logger.Error(
						nil,
						"Watch etcd registry failed: channel break, will recover from cached revision",
						"revision",
						r.currentRevision,
					)
					time.Sleep(time.Second * 5)
					etcdWatchCh = r.etcdClient.Watch(
						ctx,
						strings.TrimSuffix(r.keyPrefix, "/")+"/",
						clientv3.WithPrefix(),
						clientv3.WithPrevKV(),
						clientv3.WithRev(r.currentRevision),
					)
					break
				}

				r.logger.Debugw("etcd event trigger", "event", event)
				err := event.Err()
				if err != nil {
					switch err {
					case v3rpc.ErrCompacted, v3rpc.ErrFutureRev:
						r.logger.Error(
							event.Err(),
							"Watch etcd registry failed unrecoverable, need full sync to recover",
						)
						return

					default:
						r.logger.Error(
							event.Err(),
							"Watch etcd registry failed: other error, will recover from cached revision",
							"revision",
							r.currentRevision,
						)
						time.Sleep(time.Second * 5)
						etcdWatchCh = r.etcdClient.Watch(
							ctx,
							strings.TrimSuffix(r.keyPrefix, "/")+"/",
							clientv3.WithPrefix(),
							clientv3.WithPrevKV(),
							clientv3.WithRev(r.currentRevision),
						)
					}
					// break select
					break
				}

				// handle event
				for i, evt := range event.Events {
					switch event.Events[i].Type {
					case clientv3.EventTypePut:
						r.logger.Debugw(
							"Etcd Put events triggeres",
							"action",
							evt.Type,
							"key",
							string(evt.Kv.Key),
							"value",
							string(evt.Kv.Value),
						)
						metadata, err := r.extractResourceMetadata(string(evt.Kv.Key))
						outgoingCTX, span := tracer.NewTracer("RegistryWatch").
							Start(ctx, "RegistryWatch/EventPut", trace.WithAttributes(
								attribute.String("res.name", metadata.Name),
								attribute.String("stage", metadata.StageName),
								attribute.String("gateway", metadata.GatewayName),
								attribute.String("res.kind", metadata.Kind),
							))
						if err != nil {
							r.logger.Infow(
								"resource key is incorrect, skip it",
								"key",
								string(evt.Kv.Key),
								"value",
								string(evt.Kv.Value),
								"err",
								err,
							)
							span.AddEvent("SKIP")
							span.End()
							continue
						}

						// push to ret channel
						metadata.CTX = outgoingCTX
						retCh <- &metadata

						span.End()
					case clientv3.EventTypeDelete:
						r.logger.Debugw(
							"Etcd Delete events triggeres",
							"action",
							evt.Type,
							"key",
							string(evt.PrevKv.Key),
							"value",
							string(evt.PrevKv.Value),
						)
						metadata, err := r.extractResourceMetadata(string(evt.PrevKv.Key))
						outgoingCTX, span := tracer.NewTracer("RegistryWatch").
							Start(ctx, "RegistryWatch/EventDelete", trace.WithAttributes(
								attribute.String("res.name", metadata.Name),
								attribute.String("stage", metadata.StageName),
								attribute.String("gateway", metadata.GatewayName),
								attribute.String("res.kind", metadata.Kind),
							))
						if err != nil {
							r.logger.Infow(
								"deleted resource key is incorrect, skip it",
								"key",
								string(evt.PrevKv.Key),
								"value",
								string(evt.PrevKv.Value),
								"err",
								err,
							)
							span.AddEvent("SKIP")
							span.End()
							continue
						}

						// push to ret channel
						metadata.CTX = outgoingCTX
						retCh <- &metadata

						span.End()
					}
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

func (r *EtcdRegistryAdapter) extractResourceMetadata(key string) (ResourceMetadata, error) {
	ret := ResourceMetadata{}
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
	if len(matches) < 5 {
		r.logger.Error(nil, "Etcd key segment by slash should larger or equal to 5", "key", key)
		return ret, eris.Errorf("Etcd key segment by slash should larger or equal to 5")
	}
	ret.GatewayName = matches[len(matches)-5]
	ret.StageName = matches[len(matches)-4]
	ret.APIVersion = matches[len(matches)-3]
	ret.Kind = matches[len(matches)-2]
	ret.Name = matches[len(matches)-1]
	r.logger.Debugw("Extract resource info from etcdkey", "key", key, "resourceInfo", ret)
	return ret, nil
}

func (r *EtcdRegistryAdapter) yamlUnmarshal(
	gvk schema.GroupVersionKind,
	kv *mvccpb.KeyValue,
	obj client.Object,
) error {
	// convert yaml formated spec content into map
	temp := make(map[string]interface{})
	err := yaml.Unmarshal(kv.Value, &temp)
	if err != nil {
		return err
	}
	// serializing spec or data map into json formated string
	by, err := json.Marshal(temp)
	if err != nil {
		return err
	}
	// convert json string into object
	err = json.Unmarshal(by, obj)
	if err != nil {
		return err
	}
	obj.SetResourceVersion(strconv.FormatInt(kv.ModRevision, 10))
	return nil
}
