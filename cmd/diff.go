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

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/api/handler"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type diffCommand struct {
	cmd *cobra.Command
}

var diffCmd = &diffCommand{}

func init() {
	diffCmd.Init()
}

// Init ...
func (d *diffCommand) Init() {
	cmd := &cobra.Command{
		Use:          "diff",
		Short:        "diff between bkgateway resources and apisix storage",
		Run:          func(cmd *cobra.Command, args []string) {},
		SilenceUsage: true,
		PreRun:       preRun,
		RunE:         d.RunE,
	}

	cmd.Flags().String("gateway", "", "gateway name for list command")
	cmd.Flags().String("stage", "", "stage name for list command")
	cmd.Flags().Int64("resource_id", -1, "resource ID for list command, default(-1) for all resources in stage")
	cmd.Flags().String("resource_name", "", "resource Name for list command, empty for all resources in stage")
	cmd.Flags().StringP("write-out", "w", "simple", "response write out format (simple, json, yaml)")
	cmd.Flags().Bool("all", false, "list all gateway resources")
	cmd.MarkFlagsRequiredTogether("gateway", "stage")
	cmd.MarkFlagsMutuallyExclusive("resource_id", "resource_name")

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yml;required)")
	cmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	cmd.MarkFlagRequired("config")
	viper.SetDefault("author", "blueking-paas")

	rootCmd.AddCommand(cmd)
	d.cmd = cmd
}

// RunE ...
func (d *diffCommand) RunE(cmd *cobra.Command, args []string) error {
	initClient()

	cli, err := client.GetLeaderResourceClient(globalConfig.HttpServer.AuthPassword)
	if err != nil {
		logger.Infow("GetLeaderResourcesClient failed", "err", err)
		return err
	}
	if cli == nil {
		logger.Error(err, "GetLeaderResourcesClient nil")
		return err
	}
	req := &handler.DiffReq{}
	req.Gateway, _ = cmd.Flags().GetString("gateway")
	req.Stage, _ = cmd.Flags().GetString("stage")
	var resourceIdentity handler.ResourceInfo
	resName, _ := cmd.Flags().GetString("resource_name")
	resID, _ := cmd.Flags().GetInt64("resource_id")
	if len(resName) != 0 {
		resourceIdentity = handler.ResourceInfo{
			ResourceName: resName,
		}
	} else if resID >= 0 {
		resourceIdentity = handler.ResourceInfo{
			ResourceId: resID,
		}
	}
	req.Resource = &resourceIdentity
	req.All, _ = cmd.Flags().GetBool("all")

	if err := d.validateRequest(req); err != nil {
		return err
	}
	resp, err := cli.Diff(req)
	if err != nil {
		logger.Error(err, "Diff request failed")
		return err
	}
	format, _ := cmd.Flags().GetString("write-out")
	err = d.formatOutput(*resp, format)
	if err != nil {
		logger.Error(err, "print resp failed")
		return err
	}
	return nil
}

func (d *diffCommand) formatOutput(diffInfo handler.DiffInfo, format string) error {
	switch format {
	case "json":
		return printJson(diffInfo)
	case "yaml":
		return printYaml(diffInfo)
	case "simple":
		for stage, diffResources := range diffInfo {
			d.printResource(stage, "Route", diffResources.Routes)
			d.printResource(stage, "Service", diffResources.Services)
			d.printResource(stage, "PluginMetadata", diffResources.PluginMetadata)
			d.printResource(stage, "SSL", diffResources.Ssl)
			fmt.Println()
		}
	}
	return nil
}

func (d *diffCommand) printResource(stage, typeName string, fields map[string]interface{}) {
	if fields == nil {
		return
	}
	for id, value := range fields {
		fmt.Printf("Stage: %s, %s: %s\n%s\n", stage, typeName, id, value)
	}
}

func (d *diffCommand) validateRequest(req *handler.DiffReq) error {
	if len(req.Gateway) == 0 && !req.All {
		return eris.New("--gateway --stage, or --all should be set")
	}
	return nil
}
