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
	"fmt"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/protocol"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type syncCommand struct {
	cmd *cobra.Command
}

var syncCmd = &syncCommand{}

func init() {
	syncCmd.Init()
}

// Init ...
func (s *syncCommand) Init() {
	cmd := &cobra.Command{
		Use:          "sync",
		Short:        "sync bkgateway resources into apisix storage",
		SilenceUsage: true,
		PreRun:       preRun,
		RunE:         s.RunE,
	}

	// flags
	cmd.Flags().String("gateway", "", "gateway for sync command")
	cmd.Flags().String("stage", "", "stage for sync command")
	cmd.Flags().Bool("all", false, "sync all gateway")
	// constraints
	cmd.MarkFlagsRequiredTogether("gateway", "stage")

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yml;required)")
	cmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	cmd.MarkFlagRequired("config")
	viper.SetDefault("author", "blueking-paas")

	rootCmd.AddCommand(cmd)
	s.cmd = cmd
}

// RunE run the sync command
func (s *syncCommand) RunE(cmd *cobra.Command, args []string) error {
	initClient()

	// prepare grpc cli
	cli, err := client.GetLeaderResourceClient(globalConfig.HttpServer.ApiKey)
	if err != nil {
		logger.Infow("GetLeaderResourcesClient failed", "err", err)
		return err
	}
	if cli == nil {
		logger.Error(err, "GetLeaderResourcesClient failed")
		return err
	}

	// build request
	req := &protocol.SyncReq{}
	req.Gateway, _ = cmd.Flags().GetString("gateway")
	req.Stage, _ = cmd.Flags().GetString("stage")
	req.All, _ = cmd.Flags().GetBool("all")

	// validate
	if err := s.validateRequest(req); err != nil {
		return err
	}
	err = cli.Sync(req)
	if err != nil {
		logger.Error(err, "Sync request failed")
		return err
	}
	fmt.Println("Sync task sent")
	return nil
}

func (s *syncCommand) validateRequest(req *protocol.SyncReq) error {
	if len(req.Gateway) == 0 && !req.All {
		return eris.New("--gateway --stage, or --all should be set")
	}
	return nil
}
