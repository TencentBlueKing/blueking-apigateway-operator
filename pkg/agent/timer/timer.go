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

package timer

import (
	"sync"
	"time"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
)

// CacheTimer ...
type CacheTimer struct {
	CachedTime       time.Time
	ShouldCommitTime time.Time
}

// Reset ...
func (t *CacheTimer) Reset(offset time.Duration) {
	t.CachedTime = time.Now()
	t.ShouldCommitTime = time.Now().Add(offset)
}

// Update ...
func (t *CacheTimer) Update(offset time.Duration) {
	t.ShouldCommitTime = time.Now().Add(offset)
}

// StageTimer ...
type StageTimer struct {
	stageTimer sync.Map
}

// NewStageTimer ...
func NewStageTimer() *StageTimer {
	return &StageTimer{}
}

// Update ...
func (t *StageTimer) Update(stage registry.StageInfo) {
	var timer *CacheTimer
	timerInterface, ok := t.stageTimer.Load(stage)
	if !ok {
		timer = &CacheTimer{}
		timer.Reset(eventsWaitingTimeWindow)
	} else {
		timer = timerInterface.(*CacheTimer)

		// end old stage trace
		_, span := trace.StartTrace(timer.StageInfo.Ctx, "timer.Replace")
		span.End()

		timer.StageInfo = stage

		timer.Update(eventsWaitingTimeWindow)
	}
	t.stageTimer.Store(stage, timer)
}

// ListStagesForCommit ...
func (t *StageTimer) ListStagesForCommit() []registry.StageInfo {
	stageList := make([]registry.StageInfo, 0)

	t.stageTimer.Range(func(stageInterface, timerInterface interface{}) bool {
		stage := stageInterface.(registry.StageInfo)
		timer := timerInterface.(*CacheTimer)

		if time.Since(timer.ShouldCommitTime) > 0 || time.Since(timer.CachedTime) > forceUpdateTimeWindow {
			stageList = append(stageList, stage)
			t.stageTimer.Delete(stage)
		}
		return true
	})

	return stageList
}
