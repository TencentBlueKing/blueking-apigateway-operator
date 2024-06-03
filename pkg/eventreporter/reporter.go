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

package eventreporter

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

var (
	reporter     *Reporter
	reporterOnce sync.Once
)

type reportEvent struct {
	ctx    context.Context
	stage  *v1beta1.BkGatewayStage
	Event  constant.EventName
	status constant.EventStatus
	detail map[string]interface{}
	// event timestamp
	ts int64
}

type versionProbe struct {
	chain    chan struct{} // control version probe concurrency
	timeout  time.Duration // control version probe timeout
	waitTime time.Duration // control version probe wait time
}

type Reporter struct {
	eventChain   chan reportEvent
	reportChain  chan struct{} // control reporter concurrency
	close        chan struct{}
	versionProbe versionProbe
}

// InitReporter
func InitReporter(cfg *config.Config) {
	reporterOnce.Do(func() {
		reporter = &Reporter{
			eventChain:  make(chan reportEvent, cfg.EventReporter.EventBufferSize),
			reportChain: make(chan struct{}, cfg.EventReporter.ReporterBufferSize),
			close:       make(chan struct{}),
			versionProbe: versionProbe{
				chain:    make(chan struct{}, cfg.EventReporter.VersionProbe.BufferSize),
				timeout:  cfg.EventReporter.VersionProbe.Timeout,
				waitTime: cfg.EventReporter.VersionProbe.WaitTime,
			},
		}
	})
}

// Start reporter
func Start(ctx context.Context) {
	utils.GoroutineWithRecovery(ctx, func() {
		for event := range reporter.eventChain {
			reporter.reportChain <- struct{}{}
			// Concurrent processing to avoid processing too slow
			tempEvent := event // Avoid closure problems
			utils.GoroutineWithRecovery(ctx, func() {
				reporter.reportEvent(tempEvent)
			})
		}
		reporter.close <- struct{}{}
		logging.GetLogger().Info("reporter exiting")
	})
}

// Shutdown
// Note: Here you need to close the eventChain data source committer first,
//
//	and then close the close, otherwise writing to the eventChain will panic
func Shutdown() {
	logging.GetLogger().Info("reporter  closing")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	close(reporter.eventChain)
	select {
	case <-reporter.close:
		logging.GetLogger().Info("reporter closed")
	case <-ctx.Done():
		log.Println("close reporter timeout of 5 seconds")
	}
}

// ReportParseConfigurationDoingEvent  will report the event of paring configuration
func ReportParseConfigurationDoingEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	event := reportEvent{
		stage:  stage,
		Event:  constant.EventNameParseConfiguration,
		status: constant.EventStatusDoing,
		detail: nil,
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportParseConfigurationFailureEvent will report parse configuration failure event
func ReportParseConfigurationFailureEvent(ctx context.Context, stage *v1beta1.BkGatewayStage, err error) {
	event := reportEvent{
		ctx:    ctx,
		stage:  stage,
		Event:  constant.EventNameParseConfiguration,
		status: constant.EventStatusFailure,
		detail: map[string]interface{}{"err_msg": err.Error()},
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportParseConfigurationSuccessEvent will report the success event of parse configuration
func ReportParseConfigurationSuccessEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	event := reportEvent{
		stage:  stage,
		Event:  constant.EventNameParseConfiguration,
		status: constant.EventStatusSuccess,
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportApplyConfigurationDoingEvent will report the event of applying configuration
func ReportApplyConfigurationDoingEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	event := reportEvent{
		stage:  stage,
		Event:  constant.EventNameApplyConfiguration,
		status: constant.EventStatusDoing,
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportApplyConfigurationSuccessEvent will report success event when apply configuration successfully
func ReportApplyConfigurationSuccessEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	event := reportEvent{
		stage:  stage,
		Event:  constant.EventNameApplyConfiguration,
		status: constant.EventStatusSuccess,
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportLoadConfigurationDoingEvent will report  event when loading configuration
func ReportLoadConfigurationDoingEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	event := reportEvent{
		stage:  stage,
		Event:  constant.EventNameLoadConfiguration,
		status: constant.EventStatusDoing,
		ts:     time.Now().Unix(),
	}
	addEvent(event)
}

// ReportLoadConfigurationResultEvent Report the detection result of apisix loading
func ReportLoadConfigurationResultEvent(ctx context.Context, stage *v1beta1.BkGatewayStage) {
	// filter not need report event
	if stage == nil || stage.Labels == nil {
		return
	}
	publishID := stage.Labels[config.BKAPIGatewayLabelKeyGatewayPublishID]
	if publishID == constant.NoNeedReportPublishID || publishID == "" {
		logging.GetLogger().Debugf("event[stage: %+v] is not need to report", stage.Labels)
		return
	}

	reporter.versionProbe.chain <- struct{}{} // control concurrency
	utils.GoroutineWithRecovery(ctx, func() {
		defer func() {
			<-reporter.versionProbe.chain
		}()

		// wait apisix rebuild finished then begin version probe
		time.Sleep(reporter.versionProbe.waitTime)
		eventReq := parseEventInfo(stage)
		reportCtx, cancelFunc := context.WithTimeout(ctx, reporter.versionProbe.timeout)
		errChan := make(chan error, 1)
		defer func() {
			cancelFunc()
			close(errChan)
		}()

		// publish probe
		utils.GoroutineWithRecovery(ctx, func() {
			// begin publish probe
			versionInfo, err := client.GetApisixClient().
				GetReleaseVersion(eventReq.BkGatewayName, eventReq.BkStageName, eventReq.PublishID)
			errChan <- err
			if err != nil {
				logging.GetLogger().Errorf(
					"get release[gateway:%s,stage:%s,publish_id:%s] version from apisix err:%v",
					eventReq.BkGatewayName, eventReq.BkStageName, eventReq.PublishID, err)
				return
			}
			event := reportEvent{
				stage:  stage,
				Event:  constant.EventNameLoadConfiguration,
				status: constant.EventStatusSuccess,
				detail: map[string]interface{}{
					"publish_id": versionInfo.PublishID,
					"start_time": versionInfo.StartTime,
				},
				ts: time.Now().Unix(),
			}
			reporter.eventChain <- event
		})
		select {
		case err := <-errChan:
			if err != nil {
				event := reportEvent{
					stage:  stage,
					Event:  constant.EventNameLoadConfiguration,
					status: constant.EventStatusFailure,
					detail: map[string]interface{}{"err_msg": err.Error()},
					ts:     time.Now().Unix(),
				}
				reporter.eventChain <- event
			}
			return
		case <-reportCtx.Done():
			// version publish probe timeout
			event := reportEvent{
				stage:  stage,
				Event:  constant.EventNameLoadConfiguration,
				status: constant.EventStatusFailure,
				detail: map[string]interface{}{"err_msg": "version publish probe timeout"},
				ts:     time.Now().Unix(),
			}
			reporter.eventChain <- event
		}
	})
}

// addEvent add event to reporter event
func addEvent(event reportEvent) {
	// avoid gateway del that cause stage to be nil and make panic
	if event.stage == nil || event.stage.Labels == nil {
		return
	}
	// filter not need report event
	publishID := event.stage.Labels[config.BKAPIGatewayLabelKeyGatewayPublishID]
	if publishID == constant.NoNeedReportPublishID || publishID == "" {
		logging.GetLogger().Debugf("event[stage: %+v] is not need to report", event.stage.Labels)
		return
	}
	reporter.eventChain <- event
}

// reportEvent
func (r *Reporter) reportEvent(event reportEvent) {
	defer func() {
		<-r.reportChain
	}()
	if event.stage == nil {
		logging.GetLogger().Errorf("event[%+v]stage is empty", event)
		return
	}

	// parse event info
	eventReq := parseEventInfo(event.stage)
	eventReq.Name = event.Event
	eventReq.Status = event.status
	eventReq.Ts = event.ts
	if len(event.detail) != 0 {
		eventReq.Detail = event.detail
	}

	// report event
	err := client.GetCoreAPIClient().ReportPublishEvent(context.TODO(), eventReq)
	if err != nil && !strings.Contains(err.Error(), constant.EventDuplicatedErrMsg) {
		logging.GetLogger().Errorf(
			"report event  [name:%s,gateway:%s,stage:%s,publish_id:%s,status:%s] fail:%v",
			event.Event, eventReq.BkGatewayName, eventReq.BkStageName, eventReq.PublishID, event.status, err)
		return
	}

	// log event
	logging.GetLogger().Infof("report event [name:%s,gateway:%s,stage:%s,publish_id:%s,status:%s] success",
		event.Event, eventReq.BkGatewayName, eventReq.BkStageName, eventReq.PublishID, event.status)
}

// parseEventInfo parse stage info
func parseEventInfo(stage *v1beta1.BkGatewayStage) *client.ReportEventReq {
	gatewayName := stage.Labels[config.BKAPIGatewayLabelKeyGatewayName]
	stageName := stage.Labels[config.BKAPIGatewayLabelKeyGatewayStage]
	publishID := stage.Labels[config.BKAPIGatewayLabelKeyGatewayPublishID]
	return &client.ReportEventReq{
		BkGatewayName: gatewayName,
		BkStageName:   stageName,
		PublishID:     publishID,
	}
}
