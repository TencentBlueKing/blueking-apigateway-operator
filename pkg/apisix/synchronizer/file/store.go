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

package file

import (
	"bytes"
	"context"
	"sync"

	"github.com/natefinch/atomic"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"micro-gateway/pkg/apisix"
	"micro-gateway/pkg/apisix/synchronizer"
	"micro-gateway/pkg/logging"
)

// FileConfigStore ...
type FileConfigStore struct {
	path string // config file path
	mux  sync.Mutex

	config *apisix.ApisixConfiguration

	logger *zap.SugaredLogger
}

// NewFileConfigStore ...
func NewFileConfigStore(path string) (*FileConfigStore, error) {
	return &FileConfigStore{
		path:   path,
		logger: logging.GetLogger().Named("file-config-store"),
	}, nil
}

// Get ...
func (f *FileConfigStore) Get(stageName string) *apisix.ApisixConfiguration {
	return f.config.ExtractStagedConfiguration(stageName)
}

// GetAll ...
func (f *FileConfigStore) GetAll() map[string]*apisix.ApisixConfiguration {
	return f.config.ToStagedConfiguration()
}

// Alter ...
func (f *FileConfigStore) Alter(
	ctx context.Context,
	changedConfig map[string]*apisix.ApisixConfiguration,
	callbackFunc synchronizer.RetrySyncFunc,
) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if len(changedConfig) == 0 {
		return
	}

	fullConfig := apisix.NewEmptyApisixConfiguration()
	for _, stagedConfig := range changedConfig {
		fullConfig.MergeFrom(stagedConfig)
	}

	for stageName, config := range f.GetAll() {
		if _, ok := changedConfig[stageName]; !ok {
			fullConfig.MergeFrom(config)
		}
	}

	err := f.write(ctx, fullConfig)
	if err != nil {
		f.logger.Error(err, "Failed render config, err")
		for stageName, config := range changedConfig {
			go callbackFunc(ctx, stageName, config)
		}
		return
	}

	f.config = fullConfig // cache local config
}

func (f *FileConfigStore) write(ctx context.Context, config *apisix.ApisixConfiguration) error {
	content, err := f.marshalConfigToStandalone(config)
	if err != nil {
		f.logger.Error(err, "Failed render config, err")
		return err
	}

	err = atomic.WriteFile(f.path, bytes.NewReader(content))
	if err != nil {
		f.logger.Error(err, "write file failed", "path", f.path)
		return err
	}
	return nil
}

func (f *FileConfigStore) marshalConfigToStandalone(config *apisix.ApisixConfiguration) ([]byte, error) {
	standalone := config.ToStandalone()
	data, err := yaml.Marshal(standalone)
	if err != nil {
		f.logger.Error(err, "marshal apisix config failed")
		return nil, eris.Wrapf(err, "marshal apisix config failed")
	}
	return append(data, []byte("\n#END")...), nil
}
