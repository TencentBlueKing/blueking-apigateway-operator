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

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/serverpb"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/structpb"
)

type diffCommand struct {
	cmd *cobra.Command
}

var diffCmd = &diffCommand{}

func init() {
	diffCmd.Init()
}

// Init ...
func (c *diffCommand) Init() {
	cmd := &cobra.Command{
		Use:          "diff",
		Short:        "diff between bkgateway resources and apisix storage",
		Run:          func(cmd *cobra.Command, args []string) {},
		SilenceUsage: true,
		PreRun:       preRun,
		RunE:         c.RunE,
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
	c.cmd = cmd
}

// RunE ...
func (c *diffCommand) RunE(cmd *cobra.Command, args []string) error {
	initClient()

	client, err := client.GetLeaderResourcesClient()
	if client == nil {
		logger.Error(err, "GetLeaderResourcesClient failed")
		return err
	}
	if err != nil {
		logger.Infow("GetLeaderResourcesClient failed", "err", err)
	}
	defer client.Close()

	req := &serverpb.DiffRequest{}
	req.Gateway, _ = cmd.Flags().GetString("gateway")
	req.Stage, _ = cmd.Flags().GetString("stage")
	var resourceIdentity *serverpb.ResourceIdentity = nil
	res_name, _ := cmd.Flags().GetString("resource_name")
	res_id, _ := cmd.Flags().GetInt64("resource_id")
	if len(res_name) != 0 {
		resourceIdentity = &serverpb.ResourceIdentity{
			ResourceIdentity: &serverpb.ResourceIdentity_ResourceName{
				ResourceName: res_name,
			},
		}
	} else if res_id >= 0 {
		resourceIdentity = &serverpb.ResourceIdentity{
			ResourceIdentity: &serverpb.ResourceIdentity_ResourceId{
				ResourceId: res_id,
			},
		}
	}
	req.Resource = resourceIdentity
	req.All, _ = cmd.Flags().GetBool("all")

	if err := c.validateRequest(req); err != nil {
		return err
	}

	resp, err := client.Diff(context.Background(), req)
	if err != nil {
		logger.Error(err, "Diff request failed")
		return err
	}
	if resp.Code != 0 {
		err = eris.New(resp.Message)
		logger.Error(err, "diff failed")
		return err
	}
	format, _ := cmd.Flags().GetString("write-out")
	err = c.formatedOutput(resp, format)
	if err != nil {
		logger.Error(err, "print resp failed")
		return err
	}
	return nil
}

func (c *diffCommand) formatedOutput(resp *serverpb.DiffResponse, format string) error {
	switch format {
	case "json":
		return printJson(resp)
	case "yaml":
		return printYaml(resp)
	case "simple":
		for stage, diffResources := range resp.Data {
			c.printResource(stage, "Route", diffResources.Routes)
			c.printResource(stage, "Service", diffResources.Services)
			c.printResource(stage, "PluginMetadata", diffResources.PluginMetadata)
			c.printResource(stage, "SSL", diffResources.Ssl)
			fmt.Println()
		}
	}
	return nil
}

func (c *diffCommand) printResource(stage, typeName string, fields *structpb.Struct) {
	if fields == nil {
		return
	}
	for id, value := range fields.Fields {
		fmt.Printf("Stage: %s, %s: %s\n%s\n", stage, typeName, id, value.GetStringValue())
	}
}

func (s *diffCommand) validateRequest(req *serverpb.DiffRequest) error {
	if len(req.Gateway) == 0 && !req.All {
		return eris.New("--gateway --stage, or --all should be set")
	}
	return nil
}
