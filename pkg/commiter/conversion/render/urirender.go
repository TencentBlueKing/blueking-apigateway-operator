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

package render

import (
	"regexp"
	"strings"
)

var (
	envRegexp   *regexp.Regexp = regexp.MustCompile(`\{env.(\w+)\}`)
	paramRegexp *regexp.Regexp = regexp.MustCompile(`\{(\w+)\}`)
)

// Render ...
type Render interface {
	Render(source string, vars map[string]string) (result string)
}

// URIRender renderer /path/{env.varName}/{pathParamName}/xxx to /path/actualValue/:pathParamName/xxx
type URIRender struct{}

// GetURIRender ...
func GetURIRender() Render {
	return &URIRender{}
}

// Render ...
func (render *URIRender) Render(source string, vars map[string]string) (result string) {
	return render.paramRender(envRender(source, vars))
}

func envRender(source string, vars map[string]string) (result string) {
	allMatches := envRegexp.FindAllStringSubmatch(source, -1)
	if allMatches == nil {
		return source
	}
	for _, matches := range allMatches {
		if len(matches) != 2 {
			continue
		}
		repl, ok := vars[matches[1]]
		if !ok {
			continue
		}
		source = strings.ReplaceAll(source, matches[0], repl)
	}
	return source
}

func (render *URIRender) paramRender(source string) (result string) {
	// parameter rendering
	return paramRegexp.ReplaceAllString(source, ":$1")
}

// UpstreamURIRender renderer /path/{env.varName}/{pathParamName}/xxx to /path/actualValue/${pathParamName}/xxx
type UpstreamURIRender struct{}

// GetUpstreamURIRender ...
func GetUpstreamURIRender() Render {
	return &UpstreamURIRender{}
}

// Render ...
func (render *UpstreamURIRender) Render(source string, vars map[string]string) (result string) {
	return render.paramRender(envRender(source, vars))
}

func (render *UpstreamURIRender) paramRender(source string) (result string) {
	// parameter rendering
	return paramRegexp.ReplaceAllString(source, "$${$1}")
}
