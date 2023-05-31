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

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	//+kubebuilder:scaffold:imports

	"micro-gateway/internal/tracer"
	"micro-gateway/pkg/agent"
	"micro-gateway/pkg/apisix/synchronizer"
	"micro-gateway/pkg/client"
	"micro-gateway/pkg/commiter"
	"micro-gateway/pkg/config"
	"micro-gateway/pkg/logging"
)

var (
	rootCtx, cancel = context.WithCancel(context.Background())
	logger          *zap.SugaredLogger
)

var (
	cfgFile      string
	globalConfig *config.Config
)

// initConfig init config from args or config file
func initConfig() {
	// 0. init config
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		panic("Config file missing")
	}

	// Use config file from the flag.
	// viper.SetConfigFile(cfgFile)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Using config file: %s, read fail: err=%v", viper.ConfigFileUsed(), err))
	}
	var err error
	globalConfig, err = config.Load(viper.GetViper())
	if err != nil {
		panic(fmt.Sprintf("Could not load configurations from file, error: %v", err))
	}
}

// initLog init logger must after initConfig
func initLog() {
	logging.Init(globalConfig)
	logger = logging.GetLogger().Named("setup")
}

func initTracing() {
	if !globalConfig.Tracing.Enabled {
		return
	}
	opt := tracer.Options{
		ExporterMode:      globalConfig.Tracing.ExporterMode,
		Endpoint:          globalConfig.Tracing.Endpoint,
		URLPath:           globalConfig.Tracing.URLPath,
		BkMonitorAPMToken: globalConfig.Tracing.BkMonitorAPMToken,
	}
	err := tracer.InitTracing(rootCtx, &opt)
	if err != nil {
		logger.Error(err, "init tracing failed")
	}
}

func listenSignal() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case sig := <-c:
			logger.Infow("Got signal. Aborting...", "sig", sig)
			cancel()
			time.Sleep(time.Second * 3)
			os.Exit(1)
		case <-rootCtx.Done():
			logger.Info("root context canceled")
			time.Sleep(time.Second * 3)
			os.Exit(1)
		}
	}()
}

func preRun(cmd *cobra.Command, args []string) {
	cmd.ParseFlags(args)
	initConfig()
	initLog()
}

func initClient() {
	client.Init(globalConfig)
}

func initOperator() {
	synchronizer.Init(globalConfig)
	commiter.Init(globalConfig)
	agent.Init(globalConfig)
}
