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

package runner

import (
	"fmt"
	"os"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"

	"micro-gateway/pkg/apisix/synchronizer"
	"micro-gateway/pkg/apisix/synchronizer/etcd"
	"micro-gateway/pkg/apisix/synchronizer/file"
	"micro-gateway/pkg/config"
	"micro-gateway/pkg/logging"
	"micro-gateway/pkg/utils"
)

const (
	resourceStoreModeFile = "file"
	resourceStoreModeEtcd = "etcd"
)

func initApisixConfigStore(cfg *config.Config) (store synchronizer.ApisixConfigStore, err error) {
	switch cfg.Apisix.ResourceStoreMode {
	case resourceStoreModeEtcd:
		client, err := initApisixEtcdClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("init etcd client failed: %w", err)
		}
		store, err = etcd.NewEtcdConfigStore(client, cfg.Apisix.Etcd.KeyPrefix)
		if err != nil {
			return nil, fmt.Errorf("init etcd store failed: %w", err)
		}
		return store, nil
	case resourceStoreModeFile:
		store, err = file.NewFileConfigStore(cfg.Apisix.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("init etcd store failed: %w", err)
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unsupported resource store mode: %s", cfg.Apisix.ResourceStoreMode)
	}
}

func initOperatorEtcdClient(cfg *config.Config) (*clientv3.Client, error) {
	return createEtcdClient(&cfg.Dashboard.Etcd, cfg.Tracing.Enabled)
}

func initApisixEtcdClient(cfg *config.Config) (*clientv3.Client, error) {
	return createEtcdClient(&cfg.Apisix.Etcd, cfg.Tracing.Enabled)
}

func createEtcdClient(config *config.Etcd, tracingEnabled bool) (*clientv3.Client, error) {
	if len(config.Endpoints) == 0 {
		fmt.Println("Etcd endpoints is empty")
		os.Exit(1)
	}
	opts := make([]grpc.DialOption, 0)
	if tracingEnabled {
		opts = append(
			opts,
			grpc.WithUnaryInterceptor(
				otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
			),
		)
		opts = append(
			opts,
			grpc.WithStreamInterceptor(
				otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
			),
		)
	}
	opt := clientv3.Config{
		Endpoints:   strings.Split(config.Endpoints, ","),
		Logger:      logging.GetControllerLogger().Named("etcd"),
		DialOptions: opts,
	}
	if !config.WithoutAuth {
		cafile := config.CACert
		certfile := config.Cert
		keyfile := config.Key
		if cafile != "" && certfile != "" && keyfile != "" {
			var err error
			opt.TLS, err = utils.NewClientTLSConfig(cafile, certfile, keyfile)
			if err != nil {
				fmt.Println(err, "Create Etcd tls config failed")
				os.Exit(1)
			}
		}
		username := config.Username
		password := config.Password
		if username != "" && password != "" {
			opt.Username = username
			opt.Password = password
		}
	}
	return clientv3.New(opt)
}
