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

package commiter

import (
	"context"
	"math"
	"sync"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/cert"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/conversion"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/service"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/eventreporter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

const maxStageRetryCount = 3

var errStageNotFound = eris.Errorf("no bk gateway stage found")

// Commiter ...
type Commiter struct {
	resourceRegistry registry.Registry

	commitChan chan []registry.StageInfo

	synchronizer synchronizer.ApisixConfigSynchronizer

	// for upstream tls cert
	radixTreeGetter radixtree.RadixTreeGetter
	stageTimer      *timer.StageTimer

	// external dependency for k8s discovery service nodes
	kubeClient client.Client

	logger *zap.SugaredLogger
}

// NewCommiter 创建Commiter
func NewCommiter(
	resourceRegistry registry.Registry,
	synchronizer synchronizer.ApisixConfigSynchronizer,
	radixTreeGetter radixtree.RadixTreeGetter,
	stageTimer *timer.StageTimer,
	kubeClient client.Client,
) *Commiter {
	return &Commiter{
		resourceRegistry: resourceRegistry,
		commitChan:       make(chan []registry.StageInfo),
		synchronizer:     synchronizer,
		radixTreeGetter:  radixTreeGetter,
		stageTimer:       stageTimer,
		kubeClient:       kubeClient,
		logger:           logging.GetLogger().Named("commiter"),
	}
}

// Run ...
func (c *Commiter) Run(ctx context.Context) {
	// 分批次处理需要同步的stage
	for {
		c.logger.Debugw("commiter waiting for commit command")

		select {
		case stageList := <-c.commitChan:
			c.logger.Debugw("Commit stage changes", "stageList", stageList)

			offset := 0
			segmentLength := concurrencyLimit
			totalSegments := len(stageList) / segmentLength
			totalSegments += int(math.Ceil(float64(len(stageList)%segmentLength) / float64(segmentLength)))

			for ; offset < totalSegments-1; offset++ {
				c.commitGroup(ctx, stageList[offset*segmentLength:offset*segmentLength+segmentLength])
			}

			c.commitGroup(ctx, stageList[offset*segmentLength:])

			c.logger.Infow("Commit stage keys done", "stageList", stageList)

		case <-ctx.Done():
			c.logger.Info("gateway agent stopped, stop commit")
			return
		}
	}
}

// GetCommitChan 获取提交channel
func (c *Commiter) GetCommitChan() chan []registry.StageInfo {
	return c.commitChan
}

// ForceCommit ...
func (c *Commiter) ForceCommit(ctx context.Context, stageList []registry.StageInfo) {
	c.logger.Infow("force commit stage changes", "stageList", stageList)
	c.commitChan <- stageList
}

func (c *Commiter) commitGroup(ctx context.Context, stageInfoList []registry.StageInfo) {
	c.logger.Debugw("Commit stage group", "stageList", stageInfoList)

	// batch write apisix conf to buffer
	wg := &sync.WaitGroup{}
	for _, stageInfo := range stageInfoList {
		wg.Add(1)
		tempStageInfo := stageInfo
		utils.GoroutineWithRecovery(ctx, func() {
			c.commitStage(ctx, tempStageInfo, wg)
		})
	}
	wg.Wait()
}

func (c *Commiter) commitStage(ctx context.Context, si registry.StageInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	// trace
	_, span := trace.StartTrace(si.Ctx, "commiter.commitStage")
	defer span.End()

	span.AddEvent("commiter.ConvertEtcdKVToApisixConfiguration")
	// 每一个提交都是对一个stage的全量数据处理
	apisixConf, stage, err := c.ConvertEtcdKVToApisixConfiguration(ctx, si)
	if err != nil {
		eventreporter.ReportParseConfigurationFailureEvent(ctx, stage, err)
		if !eris.Is(err, errStageNotFound) {
			c.logger.Error(err, "convert stage resources to apisix representation failed", "stageInfo", si)
			// retry
			c.retryStage(si)

			span.RecordError(err)
			return
		}

		c.logger.Infow("stage not found, delete it", "stageInfo", si)
		// 如果stage不存在, 直接删除stage
		apisixConf = apisix.NewEmptyApisixConfiguration()

		span.AddEvent("commiter.DeleteStage")
	} else {
		eventreporter.ReportParseConfigurationSuccessEvent(ctx, stage)
	}
	eventreporter.ReportApplyConfigurationDoingEvent(ctx, stage)

	err = c.synchronizer.Sync(
		ctx,
		si.GatewayName,
		si.StageName,
		apisixConf,
	)
	if err != nil {
		c.logger.Error(err, "sync stage resources to apisix failed", "stageInfo", si)
		c.retryStage(si)
	}

	// eventrepoter.ReportApplyConfigurationSuccessEvent(ctx, stage) // 可以由事件之前的关系推断出来
	eventreporter.ReportLoadConfigurationResultEvent(ctx, stage)
}

func (c *Commiter) retryStage(si registry.StageInfo) {
	if si.RetryCount >= maxStageRetryCount {
		c.logger.Errorf("too many retries", "stageInfo", si)
		return
	}

	si.RetryCount++
	c.stageTimer.Update(si)
}

// ConvertEtcdKVToApisixConfiguration ...
func (c *Commiter) ConvertEtcdKVToApisixConfiguration(
	ctx context.Context,
	si registry.StageInfo,
) (*apisix.ApisixConfiguration, *v1beta1.BkGatewayStage, error) {
	stage, err := c.getStage(ctx, si)
	if err != nil {
		return nil, nil, err
	}
	eventreporter.ReportParseConfigurationDoingEvent(ctx, stage)
	resList, err := c.listResources(ctx, si)
	if err != nil {
		return nil, stage, err
	}
	streamResList, err := c.listStreamResources(ctx, si)
	if err != nil {
		return nil, stage, err
	}
	svcList, err := c.listServices(ctx, si)
	if err != nil {
		return nil, stage, err
	}
	sslList, err := c.listSSLs(ctx, si)
	if err != nil {
		return nil, stage, err
	}
	pluginMetadatas, err := c.listPluginMetadatas(ctx, si)
	if err != nil {
		return nil, stage, err
	}

	// 单一stage的转换
	cvt, err := conversion.NewConverter(
		config.InstanceNamespace,
		si.GatewayName,
		stage,
		&conversion.UpstreamConfig{
			CertDetectTree:           c.radixTreeGetter.Get(si),
			InternalDiscoveryPlugins: internalDiscoveryType,
			NodeDiscoverer:           service.NewKubernetesNodeDiscoverer(c.kubeClient),
			ExternalNodeDiscoverer:   service.NewRegistryExternalNodeDiscoverer(c.resourceRegistry),
		},
		&conversion.SSLConfig{
			CertFetcher: cert.NewRegistryTLSCertFetcher(c.resourceRegistry),
		},
	)
	if err != nil {
		return nil, stage, err
	}
	conf, err := cvt.Convert(ctx, resList, streamResList, svcList, sslList, pluginMetadatas)

	metric.ReportResourceCountHelper(si.GatewayName, si.StageName, conf, ReportResourceConvertedMetric)

	return conf, stage, err
}

func (c *Commiter) getStage(ctx context.Context, stageInfo registry.StageInfo) (*v1beta1.BkGatewayStage, error) {
	stageList := &v1beta1.BkGatewayStageList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, stageList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway stage failed")
	}
	if len(stageList.Items) == 0 {
		return nil, errStageNotFound
	}
	// 如果是启动全量同步，stageInfo的publish_id的会被置为NoNeedReportPublishID，
	// 这里查出crd的stage会被用于上报，由于存在历史的publish_id这里需要被覆盖重新赋值回去,避免重复上报
	if stageInfo.PublishID == constant.NoNeedReportPublishID {
		stageList.Items[0].Labels[config.BKAPIGatewayLabelKeyGatewayPublishID] = constant.NoNeedReportPublishID
	}
	return &stageList.Items[0], nil
}

func (c *Commiter) listResources(
	ctx context.Context,
	stageInfo registry.StageInfo,
) ([]*v1beta1.BkGatewayResource, error) {
	resourceList := &v1beta1.BkGatewayResourceList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, resourceList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway resource failed")
	}
	var retList = make([]*v1beta1.BkGatewayResource, 0, len(resourceList.Items))
	for ind := range resourceList.Items {
		retList = append(retList, &resourceList.Items[ind])
	}
	return retList, nil
}

func (c *Commiter) listStreamResources(
	ctx context.Context,
	stageInfo registry.StageInfo,
) ([]*v1beta1.BkGatewayStreamResource, error) {
	resourceList := &v1beta1.BkGatewayStreamResourceList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, resourceList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway stream resource failed")
	}
	var retList = make([]*v1beta1.BkGatewayStreamResource, 0, len(resourceList.Items))
	for ind := range resourceList.Items {
		retList = append(retList, &resourceList.Items[ind])
	}
	return retList, nil
}

func (c *Commiter) listServices(
	ctx context.Context,
	stageInfo registry.StageInfo,
) ([]*v1beta1.BkGatewayService, error) {
	serviceList := &v1beta1.BkGatewayServiceList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, serviceList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway service failed")
	}
	var retList = make([]*v1beta1.BkGatewayService, 0, len(serviceList.Items))
	for ind := range serviceList.Items {
		retList = append(retList, &serviceList.Items[ind])
	}
	return retList, nil
}

func (c *Commiter) listPluginMetadatas(
	ctx context.Context,
	stageInfo registry.StageInfo,
) ([]*v1beta1.BkGatewayPluginMetadata, error) {
	pluginMetadataList := &v1beta1.BkGatewayPluginMetadataList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, pluginMetadataList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway plugin_metadata failed")
	}
	var retList = make([]*v1beta1.BkGatewayPluginMetadata, 0, len(pluginMetadataList.Items))
	for ind := range pluginMetadataList.Items {
		retList = append(retList, &pluginMetadataList.Items[ind])
	}
	return retList, nil
}

func (c *Commiter) listSSLs(ctx context.Context, stageInfo registry.StageInfo) ([]*v1beta1.BkGatewayTLS, error) {
	sslList := &v1beta1.BkGatewayTLSList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, sslList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway ssl failed")
	}
	var retList = make([]*v1beta1.BkGatewayTLS, 0, len(sslList.Items))
	for ind := range sslList.Items {
		retList = append(retList, &sslList.Items[ind])
	}
	return retList, nil
}
