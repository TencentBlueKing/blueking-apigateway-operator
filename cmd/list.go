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

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/api/handler"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"
)

type listCommand struct {
	cmd *cobra.Command
}

var listCmd = &listCommand{}

func init() {
	listCmd.Init()
}

// Init ...
func (l *listCommand) Init() {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "list resources in apisix",
		SilenceUsage: true,
		PreRun:       preRun,
		RunE:         l.RunE,
	}

	cmd.Flags()
	cmd.Flags().String("gateway", "", "gateway name for list command")
	cmd.Flags().String("stage", "", "stage name for list command")
	cmd.Flags().Int64("resource_id", -1, "resource ID for list command, default(-1) for all resources in stage")
	cmd.Flags().
		String(
			"resource_name",
			"",
			"resource name for list command, empty for all resources in stage. Can not be set with resource_id simultaneously",
		)
	cmd.Flags().StringP("write-out", "w", "json", "response write out format (simple, json, yaml)")
	cmd.Flags().Bool("all", false, "list all gateway resources")
	cmd.MarkFlagsRequiredTogether("gateway", "stage")
	cmd.MarkFlagsMutuallyExclusive("resource_id", "resource_name")

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yml;required)")
	cmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	cmd.MarkFlagRequired("config")
	viper.SetDefault("author", "blueking-paas")

	rootCmd.AddCommand(cmd)
	l.cmd = cmd
}

// RunE ...
func (l *listCommand) RunE(cmd *cobra.Command, args []string) error {
	initClient()

	cli, err := client.GetLeaderResourceClient(globalConfig.HttpServer.AuthPassword)
	if err != nil {
		logger.Infow("GetLeaderResourcesClient failed", "err", err)
		return err
	}
	if cli == nil {
		logger.Error(err, "GetLeaderResourcesClient failed")
		return err
	}
	req := &handler.ListReq{}
	req.Gateway, _ = cmd.Flags().GetString("gateway")
	req.Stage, _ = cmd.Flags().GetString("stage")
	resName, _ := cmd.Flags().GetString("resource_name")
	resID, _ := cmd.Flags().GetInt64("resource_id")

	req.Resource = &handler.ResourceInfo{
		ResourceId:   resID,
		ResourceName: resName,
	}
	req.All, _ = cmd.Flags().GetBool("all")

	if err := l.validateRequest(req); err != nil {
		return err
	}
	listReq := &client.ListReq{
		Gateway: req.Gateway,
		Stage:   req.Stage,
		Resource: &client.ResourceInfo{
			ResourceId:   req.Resource.ResourceId,
			ResourceName: req.Resource.ResourceName,
		},
		All: req.All,
	}
	resp, err := cli.List(listReq)
	if err != nil {
		logger.Error(err, "List request failed")
		return err
	}
	format, _ := cmd.Flags().GetString("write-out")
	err = l.formatOutput(resp, format)
	if err != nil {
		logger.Error(err, "print resp failed")
		return err
	}
	return nil
}

func (l *listCommand) formatOutput(listInfo client.ListInfo, format string) error {
	switch format {
	case "json":
		return printJson(listInfo)
	case "yaml":
		return printYaml(listInfo)
	case "simple":
		for stage, listResources := range listInfo {
			fmt.Printf("Stage: %s\n", stage)
			l.printResource("Routes", listResources.Routes)
			l.printResource("Services", listResources.Services)
			l.printResource("PluginMetadatas", listResources.PluginMetadata)
			l.printResource("SSLs", listResources.Ssl)
		}
	}
	return nil
}

func (l *listCommand) printResource(typeName string, fields map[string]interface{}) {
	fmt.Printf("\t%s:\n", typeName)
	if fields == nil {
		return
	}
	for id := range fields {
		fmt.Printf("\t\t%s\n", id)
	}
}

func (l *listCommand) validateRequest(req *handler.ListReq) error {
	if len(req.Gateway) == 0 && !req.All {
		return eris.New("--gateway --stage, or --all should be set")
	}
	return nil
}
