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

package radixtree

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// RadixTreeGetter ...
type RadixTreeGetter interface {
	Get(stage registry.StageInfo) RadixTree
	RemoveNotExistStage(existStageList []registry.StageInfo)
}

// SingleRadixTreeGetter ...
type SingleRadixTreeGetter struct {
	tree RadixTree
}

// NewSingleRadixTreeGetter ...
func NewSingleRadixTreeGetter() RadixTreeGetter {
	return &SingleRadixTreeGetter{
		tree: NewSuffixRadixTree(),
	}
}

// Get ...
func (t *SingleRadixTreeGetter) Get(_ registry.StageInfo) RadixTree {
	return t.tree
}

// RemoveNotExistStage ...
func (t *SingleRadixTreeGetter) RemoveNotExistStage(_ []registry.StageInfo) {}

// StageRadixTreeGetter ...
type StageRadixTreeGetter struct {
	stageTree map[registry.StageInfo]RadixTree
}

// NewStageRadixTreeGetter ...
func NewStageRadixTreeGetter() RadixTreeGetter {
	return &StageRadixTreeGetter{
		stageTree: make(map[registry.StageInfo]RadixTree),
	}
}

// Get ...
func (t *StageRadixTreeGetter) Get(stage registry.StageInfo) RadixTree {
	if _, ok := t.stageTree[stage]; !ok {
		t.stageTree[stage] = NewSuffixRadixTree()
	}

	return t.stageTree[stage]
}

// RemoveNotExistStage ...
func (t *StageRadixTreeGetter) RemoveNotExistStage(existStageList []registry.StageInfo) {
	existStageSet := make(map[registry.StageInfo]struct{}, len(existStageList))
	for _, existStage := range existStageList {
		existStageSet[existStage] = struct{}{}
	}

	for stage := range t.stageTree {
		if _, ok := existStageSet[stage]; !ok {
			delete(t.stageTree, stage)
		}
	}
}
