/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 微网关(BlueKing - Micro APIGateway) available.
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

// Package schema ...
package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap/buffer"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils/sslx"
)

// 定义允许的操作符白名单
var allowedOps = map[string]bool{
	"==":  true, // 等于
	"~=":  true, // 不等于
	">":   true, // 大于
	"<":   true, // 小于
	"~~":  true, // 正则匹配
	"~*":  true, // 不区分大小写的正则匹配
	"IN":  true, // 在
	"HAS": true, // 包含
}

// FuncGetCustomSchema ...
type FuncGetCustomSchema func(ctx context.Context, name string) map[string]interface{}

// APISIXValidateError ...
type APISIXValidateError struct {
	Err error
}

// Validator ...
type Validator interface {
	Validate(obj json.RawMessage) error
}

// APISIXJsonSchemaValidator ...
type APISIXJsonSchemaValidator struct {
	schema                   *gojsonschema.Schema
	schemaDef                string
	version                  constant.APISIXVersion
	resourceType             constant.APISIXResource
	customizePluginSchemaMap map[string]interface{}
}

// NewResourceSchema 获取资源 schema
func NewResourceSchema(
	version constant.APISIXVersion,
	resourceType constant.APISIXResource,
	jsonPath string,
) (string, *gojsonschema.Schema, error) {
	schemaDef := schemaVersionMap[version].Get(jsonPath).String()
	if schemaDef == "" {
		log.Warnf("schema validate failed: schema not found, path: %s", jsonPath)
		return "", nil, fmt.Errorf("schema 验证失败: 未找到 schema, 路径: %s", jsonPath)
	}

	if resourceType == constant.PluginMetadata {
		// 允许有附加属性
		schema, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(schemaDef))
		if err != nil {
			log.Warnf("new schema failed: %v", err)
			return "", nil, fmt.Errorf("实例化 schema 失败: %w", err)
		}
		return schemaDef, schema, nil
	}
	// 不允许有额外字段，需动态设置 additionalProperties=false
	jsonLoader := gojsonschema.NewStringLoader(schemaDef)
	schemaObj, err := jsonLoader.LoadJSON()
	if err != nil {
		log.Warnf("schema validate failed: schema json decode failed, path: %s, %v", jsonPath, err)
		return "", nil, fmt.Errorf("schema 验证失败: schema json decode 失败, 路径: %s, %v", jsonPath, err)
	}
	schemaMap, ok := schemaObj.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("schema 验证失败: schema 不是有效的对象类型, 路径: %s", jsonPath)
	}
	schemaMap["additionalProperties"] = false
	schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(schemaMap))
	if err != nil {
		log.Warnf("new schema failed: %v", err)
		return "", nil, fmt.Errorf("实例化 schema 失败: %w", err)
	}
	return schemaDef, schema, nil
}

// NewAPISIXJsonSchemaValidator 创建 APISIXJsonSchemaValidator
func NewAPISIXJsonSchemaValidator(version constant.APISIXVersion,
	resourceType constant.APISIXResource, jsonPath string,
) (Validator, error) {
	schemaDef, schema, err := NewResourceSchema(version, resourceType, jsonPath)
	if err != nil {
		return nil, err
	}
	return &APISIXJsonSchemaValidator{
		schema:       schema,
		schemaDef:    schemaDef,
		version:      version,
		resourceType: resourceType,
	}, nil
}

func getPlugins(reqBody interface{}) (map[string]interface{}, string) {
	switch bodyType := reqBody.(type) {
	case *entity.Route:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "schema"
	case *entity.Service:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "schema"
	case *entity.Consumer:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "consumer_schema"
	case *entity.ConsumerGroup:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "consumer_schema"
	case *entity.PluginConfig:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "schema"
	case *entity.GlobalRule:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "schema"
	case *entity.StreamRoute:
		log.Infof("type of reqBody: %#v", bodyType)
		return bodyType.Plugins, "stream_schema"
	case *entity.PluginMetadata:
		log.Infof("type of reqBody: %#v", bodyType)
		name := cast.ToString(bodyType.PluginMetadataConf["name"])
		return map[string]interface{}{name: map[string]interface{}(bodyType.PluginMetadataConf)}, "metadata_schema"
	}
	return nil, ""
}

func (v *APISIXJsonSchemaValidator) cHashKeySchemaCheck(upstream *entity.UpstreamDef) error {
	if upstream.HashOn == "consumer" {
		return nil
	}
	if upstream.HashOn != "vars" &&
		upstream.HashOn != "header" &&
		upstream.HashOn != "cookie" {
		return fmt.Errorf("无效的哈希类型: %s", upstream.HashOn)
	}

	var schemaDef string
	if upstream.HashOn == "vars" {
		schemaDef = schemaVersionMap[v.version].Get("main.upstream_hash_vars_schema").String()
		if schemaDef == "" {
			return fmt.Errorf("schema 验证失败: 未找到 schema, 路径: main.upstream_hash_vars_schema")
		}
	}

	if upstream.HashOn == "header" || upstream.HashOn == "cookie" {
		schemaDef = schemaVersionMap[v.version].Get("main.upstream_hash_header_schema").String()
		if schemaDef == "" {
			return fmt.Errorf("schema 验证失败: 未找到 schema, 路径: main.upstream_hash_header_schema")
		}
	}

	s, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(schemaDef))
	if err != nil {
		return fmt.Errorf("schema 验证失败: %s", err)
	}

	ret, err := s.Validate(gojsonschema.NewGoLoader(upstream.Key))
	if err != nil {
		return fmt.Errorf("schema 验证失败: %s", err)
	}

	if !ret.Valid() {
		errString := GetSchemaValidateFailed(ret)
		return fmt.Errorf("schema 验证失败: %s", errString)
	}

	return nil
}

func (v *APISIXJsonSchemaValidator) checkUpstream(upstream *entity.UpstreamDef) error {
	if upstream == nil {
		return nil
	}

	if upstream.PassHost == "node" && upstream.Nodes != nil {
		nodes, ok := entity.NodesFormat(upstream.Nodes).([]*entity.Node)
		if !ok {
			return fmt.Errorf("当 `pass_host` 为 `node` 时, upstreams 节点不支持值 %v", nodes)
		} else if len(nodes) != 1 {
			return fmt.Errorf("当 `pass_host` 为 `node` 时, 目前仅支持 `node` 模式下的单节点")
		}
	}

	if upstream.PassHost == "rewrite" && upstream.UpstreamHost == "" {
		return fmt.Errorf("`当 `pass_host` 为 `rewrite` 时, `upstream_host` 不可为空")
	}

	// check upstream ssl
	if upstream.TLS != nil && (upstream.TLS.ClientCert != "" || upstream.TLS.ClientKey != "") {
		_, err := sslx.ParseCert(upstream.TLS.ClientCert, upstream.TLS.ClientKey)
		if err != nil {
			return err
		}
		_, err = sslx.X509CertValidity(upstream.TLS.ClientCert)
		if err != nil {
			return err
		}
	}

	if upstream.Type != "chash" {
		return nil
	}

	// to confirm
	if upstream.HashOn == "" {
		upstream.HashOn = "vars"
	}

	if upstream.HashOn != "consumer" && upstream.Key == "" {
		return fmt.Errorf("缺少键")
	}

	if err := v.cHashKeySchemaCheck(upstream); err != nil {
		return err
	}

	return nil
}

func checkRemoteAddr(remoteAddrs []string) error {
	for _, remoteAddr := range remoteAddrs {
		if remoteAddr == "" {
			return fmt.Errorf("schema 验证失败: 无效字段 remote_addrs")
		}
	}
	return nil
}

// validateVarItem 校验单个 var 条目
func validateVarItem(item []interface{}) error {
	length := len(item)
	// 检查数组长度
	if length != 3 && length != 4 {
		return errors.New("var 项必须为三元组或四元组")
	}
	// 检查变量名是否为字符串
	if _, ok := item[0].(string); !ok {
		return errors.New("变量名必须为字符串")
	}
	// 处理四元组 [!]
	if length == 4 {
		// 第二个元素必须是 "!"
		if negate, ok := item[1].(string); !ok || negate != "!" {
			return errors.New("四元组第二位必须为 '!'")
		}
		// 检查第三位是否为合法操作符
		if op, ok := item[2].(string); !ok || !allowedOps[op] {
			return errors.New("非法的操作符")
		}
		// 检查第四位是否存在(值校验可扩展)
		if item[3] == nil {
			return errors.New("匹配值不能为空")
		}
		return nil
	}
	// 处理三元组
	if op, ok := item[1].(string); !ok || !allowedOps[op] {
		return errors.New("非法的操作符")
	}
	// 检查值是否存在
	if item[2] == nil {
		return errors.New("匹配值不能为空")
	}
	return nil
}

// checkVars 校验 vars
func checkVars(vars []interface{}) error {
	if len(vars) == 0 {
		return nil
	}
	for i, item := range vars {
		// 检查是否为数组
		if _, ok := item.([]interface{}); !ok {
			return errors.New(" vars数组的值对象必须也是列表")
		}
		if err := validateVarItem(item.([]interface{})); err != nil {
			return fmt.Errorf("第 %d 项错误: %v", i+1, err)
		}
	}
	return nil
}

func (v *APISIXJsonSchemaValidator) checkConf(reqBody interface{}) error {
	switch bodyType := reqBody.(type) {
	case *entity.Route:
		route := reqBody.(*entity.Route)
		log.Infof("type of reqBody: %#v", bodyType)
		if err := v.checkUpstream(route.Upstream); err != nil {
			return err
		}
		// todo: this is a temporary method, we'll drop it later
		if err := checkRemoteAddr(route.RemoteAddrs); err != nil {
			return err
		}
		// check vars
		if err := checkVars(route.Vars); err != nil {
			return err
		}

	case *entity.Service:
		service := reqBody.(*entity.Service)
		if err := v.checkUpstream(service.Upstream); err != nil {
			return err
		}
	case *entity.Upstream:
		upstream := reqBody.(*entity.Upstream)
		if err := v.checkUpstream(&upstream.UpstreamDef); err != nil {
			return err
		}
		if upstream.TLS != nil && (upstream.TLS.ClientCert != "" || upstream.TLS.ClientKey != "") {
			_, err := sslx.ParseCert(upstream.TLS.ClientCert, upstream.TLS.ClientKey)
			if err != nil {
				return err
			}
			_, err = sslx.X509CertValidity(upstream.TLS.ClientCert)
			if err != nil {
				return err
			}
		}
	case *entity.SSL:
		_, err := sslx.ParseCert(bodyType.Cert, bodyType.Key)
		if err != nil {
			return err
		}
		_, err = sslx.X509CertValidity(bodyType.Cert)
		if err != nil {
			return err
		}
	}
	return nil
}

// Validate 验证
func (v *APISIXJsonSchemaValidator) Validate(rawConfig json.RawMessage) error { //nolint:gocyclo
	resourceIdentification := GetResourceIdentification(rawConfig)
	ret, err := v.schema.Validate(gojsonschema.NewBytesLoader(rawConfig))
	if err != nil {
		log.Errorf("schema validate failed: %s, s: %v, obj: %s", err, v.schema, rawConfig)
		return fmt.Errorf("资源: %s schema 验证失败: %s", resourceIdentification, err)
	}

	if !ret.Valid() {
		errString := GetSchemaValidateFailed(ret)
		log.Errorf("schema validate failed:s: %v, obj: %s", v.schemaDef, rawConfig)
		return fmt.Errorf("资源: %s schema 验证失败: %s", resourceIdentification, errString)
	}

	// custom check
	var obj interface{}
	switch v.resourceType {
	case constant.Route:
		obj = &entity.Route{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.Service:
		obj = &entity.Service{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.Upstream:
		obj = &entity.Upstream{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.PluginConfig:
		obj = &entity.PluginConfig{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.Consumer:
		obj = &entity.Consumer{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.ConsumerGroup:
		obj = &entity.ConsumerGroup{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.GlobalRule:
		obj = &entity.GlobalRule{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.PluginMetadata:
		obj = &entity.PluginMetadata{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.SSL:
		obj = &entity.SSL{}
		_ = json.Unmarshal(rawConfig, obj)
	case constant.StreamRoute:
		obj = &entity.StreamRoute{}
		_ = json.Unmarshal(rawConfig, obj)
	}
	if err := v.checkConf(obj); err != nil {
		return err
	}

	plugins, schemaType := getPlugins(obj)
	// 判断插件是否为空
	if constant.PluginsMustResourceMap[v.resourceType] && len(plugins) == 0 {
		log.Error("schema validate failed: plugins is empty")
		return fmt.Errorf("资源: %s schema 验证失败: 插件为空", resourceIdentification)
	}

	for pluginName, pluginConf := range plugins {
		var schemaMap map[string]interface{}
		schemaValue := GetPluginSchema(v.version, pluginName, schemaType)
		// 查询自定义插件
		if schemaValue == nil && v.customizePluginSchemaMap != nil {
			schemaValue = v.customizePluginSchemaMap[pluginName]
		}
		if schemaValue == nil {
			log.Errorf("schema validate failed: schema not found,  %s, %s", "plugins."+pluginName, schemaType)
			return fmt.Errorf("资源:%s schema 验证失败: 未找到 schema, 路径: %s",
				resourceIdentification, "plugins."+pluginName)
		}
		schemaMap = schemaValue.(map[string]interface{})
		schemaByte, err := json.Marshal(schemaMap)
		if err != nil {
			log.Warnf("schema validate failed: schema json encode failed, path: %s, %v", "plugins."+pluginName, err)
			return fmt.Errorf(
				"资源: %s schema 验证失败: schema json encode 失败, 路径: %s, %v",
				resourceIdentification, "plugins."+pluginName,
				err,
			)
		}

		s, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schemaByte))
		if err != nil {
			log.Errorf("init schema[pluginName:%s] validate failed: %s", pluginName, err)
			return fmt.Errorf("资源:%s 插件:%s schema 验证失败: %s", resourceIdentification, pluginName,
				err)
		}

		// check property disable, if is bool, remove from json schema checking
		conf := pluginConf.(map[string]interface{})
		var exchange bool
		disable, ok := conf["disable"]
		if ok {
			if fmt.Sprintf("%T", disable) == "bool" {
				delete(conf, "disable")
				exchange = true
			}
		}

		// check schema
		ret, err := s.Validate(gojsonschema.NewGoLoader(conf))
		if err != nil {
			log.Errorf("schema validate failed: %s", err)
			return fmt.Errorf("资源:%s 插件:%s schema 验证失败: %s", resourceIdentification, pluginName,
				err)
		}

		// put the value back to the property disable
		if exchange {
			conf["disable"] = disable
		}

		if !ret.Valid() {
			errString := GetSchemaValidateFailed(ret)
			log.Errorf("schema validate failed:s: %v, obj: %#v", v.schemaDef, rawConfig)
			return fmt.Errorf("资源:%s 插件:%s schema 验证失败: %s", resourceIdentification, pluginName,
				errString)
		}
	}

	return nil
}

// APISIXSchemaValidator ...
type APISIXSchemaValidator struct {
	schema  *gojsonschema.Schema
	version constant.APISIXVersion
}

// NewAPISIXSchemaValidator 创建 APISIXSchemaValidator
func NewAPISIXSchemaValidator(version constant.APISIXVersion, jsonPath string) (Validator, error) {
	schemaDef := schemaVersionMap[version].Get(jsonPath).String()
	if schemaDef == "" {
		log.Warnf("schema validate failed: schema not found, path: %s", jsonPath)
		return nil, fmt.Errorf("schema 验证失败: 未找到 schema, 路径: %s", jsonPath)
	}

	s, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(schemaDef))
	if err != nil {
		log.Warnf("new schema failed: %v", err)
		return nil, fmt.Errorf("实例化 schema 失败: %w", err)
	}
	return &APISIXSchemaValidator{
		schema:  s,
		version: version,
	}, nil
}

// Validate 验证
func (v *APISIXSchemaValidator) Validate(obj json.RawMessage) error {
	resourceIdentification := GetResourceIdentification(obj)
	ret, err := v.schema.Validate(gojsonschema.NewBytesLoader(obj))
	if err != nil {
		log.Warnf("resource: %s schema validate failed: %v", resourceIdentification, err)
		return fmt.Errorf("schema 验证失败: %w", err)
	}

	if !ret.Valid() {
		errString := GetSchemaValidateFailed(ret)
		return fmt.Errorf("资源: %s schema 验证失败: %s", resourceIdentification, errString)
	}

	return nil
}

// GetResourceIdentification 获取资源标识
func GetResourceIdentification(config json.RawMessage) string {
	id := gjson.GetBytes(config, "id").String()
	if id != "" {
		return id
	}
	name := gjson.GetBytes(config, "name").String()
	if name != "" {
		return name
	}
	return gjson.GetBytes(config, "username").String()
}

// GetSchemaValidateFailed 获取 schema 验证失败的错误信息
func GetSchemaValidateFailed(ret *gojsonschema.Result) string {
	errString := buffer.Buffer{}
	for i, vErr := range ret.Errors() {
		if i != 0 {
			errString.AppendString("\n")
		}
		errString.AppendString(vErr.String())
	}
	return errString.String()
}
