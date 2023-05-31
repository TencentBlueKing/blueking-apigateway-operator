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

	"micro-gateway/api/v1beta1"
	"micro-gateway/pkg/agent/timer"
	"micro-gateway/pkg/apisix"
	"micro-gateway/pkg/apisix/synchronizer"
	"micro-gateway/pkg/commiter/cert"
	"micro-gateway/pkg/commiter/conversion"
	"micro-gateway/pkg/commiter/service"
	"micro-gateway/pkg/config"
	"micro-gateway/pkg/logging"
	"micro-gateway/pkg/metric"
	"micro-gateway/pkg/radixtree"
	"micro-gateway/pkg/registry"
)

var errStageNotFound = eris.Errorf("no bk gateway stage found")

// Commiter ...
type Commiter struct {
	resourceRegistry registry.Registry

	commitChan chan []registry.StageInfo

	synchronizer *synchronizer.ApisixConfigurationSynchronizer

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
	synchronizer *synchronizer.ApisixConfigurationSynchronizer,
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
		go c.commitStage(ctx, stageInfo, wg)
	}
	wg.Wait()

	// flush all buffed apisix conf to etcd/file
	c.synchronizer.Flush(ctx)
}

func (c *Commiter) commitStage(ctx context.Context, si registry.StageInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	// 每一个提交都是对一个stage的全量数据处理
	apisixConf, err := c.ConvertEtcdKVToApisixConfiguration(ctx, si)
	if err != nil {
		if !eris.Is(err, errStageNotFound) {
			c.logger.Error(err, "convert stage resources to apisix representation failed", "stageInfo", si)
			// retry
			c.stageTimer.Update(si)
			return
		}

		c.logger.Infow("stage not found, delete it", "stageInfo", si)
		// 如果stage不存在, 直接删除stage
		apisixConf = apisix.NewEmptyApisixConfiguration()
	}

	c.synchronizer.Sync(
		ctx,
		si.GatewayName,
		si.StageName,
		apisixConf,
	)
}

// ConvertEtcdKVToApisixConfiguration ...
func (c *Commiter) ConvertEtcdKVToApisixConfiguration(
	ctx context.Context,
	si registry.StageInfo,
) (*apisix.ApisixConfiguration, error) {
	stage, err := c.getStage(ctx, si)
	if err != nil {
		return nil, err
	}
	resList, err := c.listResources(ctx, si)
	if err != nil {
		return nil, err
	}
	svcList, err := c.listServices(ctx, si)
	if err != nil {
		return nil, err
	}
	sslList, err := c.listSSLs(ctx, si)
	if err != nil {
		return nil, err
	}
	pluginMetadatas, err := c.listPluginMetadatas(ctx, si)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	conf, err := cvt.Convert(ctx, resList, svcList, sslList, pluginMetadatas)

	metric.ReportResourceCountHelper(si.GatewayName, si.StageName, conf, ReportResourceConvertedMetric)

	return conf, err
}

func (c *Commiter) getStage(ctx context.Context, stageInfo registry.StageInfo) (*v1beta1.BkGatewayStage, error) {
	stageList := &v1beta1.BkGatewayStageList{}
	if err := c.resourceRegistry.List(ctx, registry.ResourceKey{StageInfo: stageInfo}, stageList); err != nil {
		return nil, eris.Wrapf(err, "list bkgateway stage failed")
	}
	if len(stageList.Items) == 0 {
		return nil, errStageNotFound
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
	var retList []*v1beta1.BkGatewayResource
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
	var retList []*v1beta1.BkGatewayService
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
	var retList []*v1beta1.BkGatewayPluginMetadata
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
	var retList []*v1beta1.BkGatewayTLS
	for ind := range sslList.Items {
		retList = append(retList, &sslList.Items[ind])
	}
	return retList, nil
}
