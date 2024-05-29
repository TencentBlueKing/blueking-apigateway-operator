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

package conversion

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/conversion/render"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	frametypes "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

const (
	// pluginNameBKProxyRewrite plugin name for proxy rewrite
	pluginNameBKProxyRewrite = "bk-proxy-rewrite"

	passHostPass    = "pass"
	passHostNode    = "node"
	passHostRewrite = "rewrite"
)

// MATCH_SUB_PATH_PRIORITY ...
const MATCH_SUB_PATH_PRIORITY = -1000

func calculateMatchSubPathRoutePriority(path string) int {
	// 使用 / 对路径进行切分
	parts := strings.Split(path, "/")
	// 遍历切分后的路径，替换冒号开头的变量
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "a"
		}
	}
	// 使用 / 将替换后的路径拼接起来
	replacedPath := strings.Join(parts, "/")

	// the priority of subpath = -1000 + len(replacedPath)
	return len(replacedPath) + MATCH_SUB_PATH_PRIORITY
}

// convertResource convert bk gateway resource to route
func (c *Converter) convertResource(
	resource *v1beta1.BkGatewayResource,
	services []*v1beta1.BkGatewayService,
) (*apisix.Route, error) {
	var err error
	if resource == nil {
		return nil, eris.Errorf("resource or resource.spec.http cannot be empty")
	}
	newRoute := &apisix.Route{
		Route: apisixv1.Route{
			Metadata: apisixv1.Metadata{
				ID:     c.getID(resource.Spec.ID.String(), getObjectName(resource.GetName(), resource.GetNamespace())),
				Name:   getObjectName(resource.GetName(), resource.GetNamespace()),
				Desc:   resource.Spec.Desc,
				Labels: c.getLabel(),
			},
			Host:            c.stage.Spec.Domain,
			EnableWebsocket: resource.Spec.EnableWebsocket,
		},
		Status: utils.IntPtr(1),
	}
	// NOTE: labels的resource name 超过64不能被apisix从etcd读取, 没有写入
	// resName, err := c.getResourceName(resource.Spec.Name, resource.Labels)
	// if err != nil {
	// 	logger.Info("resource name not found", "resource", resource.Name, "labels", newRoute.Metadata.Labels)
	// } else {
	// 	newRoute.Metadata.Labels[config.BKAPIGatewayLabelKeyResourceName] = resName
	// }
	uriWithoutSuffixSlash := c.getOptionalUriGatewayPrefix() +
		strings.TrimSuffix(
			strings.TrimSuffix(c.stage.Spec.PathPrefix, "/")+"/"+strings.TrimPrefix(resource.Spec.URI, "/"),
			"/",
		)

	uriWithoutSuffixSlash = render.GetURIRender().Render(uriWithoutSuffixSlash, c.stage.Spec.Vars)
	// to enable prefix match, apisix.router.http = radixtree_uri should be set in config.yaml
	if resource.Spec.MatchSubPath {
		newRoute.Uris = []string{
			uriWithoutSuffixSlash,
			uriWithoutSuffixSlash + "/*" + config.BKAPIGatewaySubpathMatchParamName,
		}
		newRoute.Priority = calculateMatchSubPathRoutePriority(uriWithoutSuffixSlash)
	} else {
		pathParamInd := strings.Index(uriWithoutSuffixSlash, "/:")
		if pathParamInd != -1 {
			newRoute.Uri = uriWithoutSuffixSlash + "/?"
		} else {
			newRoute.Uris = []string{uriWithoutSuffixSlash, uriWithoutSuffixSlash + "/"}
		}
	}

	if len(resource.Spec.Methods) != 0 {
		newRoute.Methods = resource.Spec.Methods
	}

	if resource.Spec.Timeout != nil {
		newRoute.Timeout = c.convertHTTPTimeout(resource.Spec.Timeout)
	}

	pluginsMap := make(map[string]interface{})
	if resource.Spec.Rewrite != nil && resource.Spec.Rewrite.Enabled {
		pluginsMap, err = c.getProxyRewrite(resource.Spec.Rewrite, resource)
		if err != nil {
			return nil, err
		}
	}

	if len(resource.Spec.Plugins) != 0 {
		for _, p := range resource.Spec.Plugins {
			pluginName, pluginConfig := c.convertPlugin(p)
			pluginsMap[pluginName] = pluginConfig
		}
	}

	if len(pluginsMap) != 0 {
		newRoute.Plugins = pluginsMap
	}

	if resource.Spec.Upstream != nil {
		newRoute.Upstream, err = c.convertUpstream(resource.TypeMeta, resource.ObjectMeta, resource.Spec.Upstream)
		if err != nil {
			return nil, eris.Wrapf(err, "convert upstream of resource %s/%s failed",
				resource.GetName(), resource.GetNamespace())
		}
	}

	if len(resource.Spec.Service) != 0 {
		for _, svc := range services {
			if svc.Name == resource.Spec.Service {
				newRoute.ServiceID = c.getID(svc.Spec.ID, getObjectName(svc.Name, svc.Namespace))
				break
			}
		}

		if len(newRoute.ServiceID) == 0 {
			c.logger.Error(nil, "No such service match routes requirement", "resource", resource.ObjectMeta)
			newRoute.ServiceID = resource.Spec.Service
		}
	}

	c.logger.Debugw("convert resource to route", "resource", resource, "route", newRoute)

	return newRoute, nil
}

// convertStreamResource convert bk gateway stream resource to stream route
func (c *Converter) convertStreamResource(
	resource *v1beta1.BkGatewayStreamResource,
	services []*v1beta1.BkGatewayService,
) (*apisix.StreamRoute, error) {
	var err error
	if resource == nil {
		return nil, eris.Errorf("resource or resource.spec.http cannot be empty")
	}
	newRoute := &apisix.StreamRoute{
		Metadata: apisixv1.Metadata{
			ID:     c.getID(resource.Spec.ID.String(), getObjectName(resource.GetName(), resource.GetNamespace())),
			Name:   getObjectName(resource.GetName(), resource.GetNamespace()),
			Desc:   resource.Spec.Desc,
			Labels: c.getLabel(),
		},
		ServerAddr: resource.Spec.ServerAddr,
		RemoteAddr: resource.Spec.RemoteAddr,
		ServerPort: resource.Spec.ServerPort,
		SNI:        resource.Spec.SNI,
		Status:     utils.IntPtr(1),
	}

	if resource.Spec.Upstream != nil {
		newRoute.Upstream, err = c.convertUpstream(resource.TypeMeta, resource.ObjectMeta, resource.Spec.Upstream)
		if err != nil {
			return nil, eris.Wrapf(err, "convert upstream of stream resource %s/%s failed",
				resource.GetName(), resource.GetNamespace())
		}
	}

	if len(resource.Spec.Service) != 0 {
		for _, svc := range services {
			if svc.Name == resource.Spec.Service {
				newRoute.ServiceID = c.getID(svc.Spec.ID, getObjectName(svc.Name, svc.Namespace))
				break
			}
		}

		if len(newRoute.ServiceID) == 0 {
			c.logger.Error(nil, "No such service match routes requirement", "stream resource", resource.ObjectMeta)
			newRoute.ServiceID = resource.Spec.Service
		}
	}

	c.logger.Debugw("convert stream resource to route", "resource", resource, "route", newRoute)

	return newRoute, nil
}

func (c *Converter) convertPlugin(plugin *v1beta1.BkGatewayPlugin) (string, map[string]interface{}) {
	pluginName := plugin.Name
	if len(plugin.Config.Raw) == 0 {
		return pluginName, make(map[string]interface{})
	}

	tmpMap := make(map[string]interface{})
	if err := json.Unmarshal(plugin.Config.Raw, &tmpMap); err != nil {
		c.logger.Error(err, "decode plugin config failed", "plugin name", pluginName)
		return pluginName, make(map[string]interface{})
	}

	return pluginName, tmpMap
}

//nolint:gocyclo
func (c *Converter) convertUpstream(
	typeMeta metav1.TypeMeta,
	objMeta metav1.ObjectMeta,
	upstream *v1beta1.BkGatewayUpstreamConfig,
) (*apisix.Upstream, error) {
	retUpstream := &apisix.Upstream{
		Checks: upstream.Checks.ConvertToAPISIXv1Check(),
	}

	if len(upstream.Type) != 0 {
		retUpstream.Type = utils.StringPtr(upstream.Type)
	}

	if len(upstream.HashOn) != 0 {
		retUpstream.HashOn = utils.StringPtr(upstream.HashOn)
	}

	if len(upstream.Key) != 0 {
		retUpstream.Key = utils.StringPtr(upstream.Key)
	}

	if len(upstream.Scheme) != 0 {
		retUpstream.Scheme = utils.StringPtr(upstream.Scheme)
	}

	if len(upstream.PassHost) != 0 {
		retUpstream.PassHost = utils.StringPtr(upstream.PassHost)
	}

	if len(upstream.UpstreamHost) != 0 {
		retUpstream.UpstreamHost = utils.StringPtr(upstream.UpstreamHost)
	}

	if upstream.Retries != 0 {
		retUpstream.Retries = utils.IntPtr(upstream.Retries)
	}

	if upstream.RetryTimeout != nil {
		retUpstream.RetryTimeout = upstream.RetryTimeout
	}

	if upstream.Timeout != nil {
		retUpstream.Timeout = c.convertHTTPTimeout(upstream.Timeout)
	}

	switch {
	case len(upstream.Nodes) != 0:
		retUpstream.Nodes = upstream.Nodes
	case len(upstream.DiscoveryType) != 0:
		switch {
		case utils.StringInSlice(upstream.DiscoveryType, c.upstreamConfig.InternalDiscoveryPlugins):
			// 如果使用内部的发现方式, 不需要直接写入nodes
			retUpstream.DiscoveryType = utils.StringPtr(upstream.DiscoveryType)
		case upstream.DiscoveryType == "kubernetes":
			newNodes, err := c.convertKubernetesServiceNodes(objMeta.Namespace, upstream.ServiceName)
			if err != nil {
				return nil, eris.Wrapf(err, "discovery kubernetes service failed")
			}
			retUpstream.Nodes = newNodes
		default:
			return nil, eris.Errorf("unsupported discovery type %s", upstream.DiscoveryType)
		}
	case len(upstream.ExternalDiscoveryType) != 0:
		newNodes, err := c.externalExternalServiceNodes(typeMeta, objMeta, upstream.ExternalDiscoveryType)
		if err != nil {
			return nil, eris.Wrapf(err, "discovery external service failed")
		}
		retUpstream.Nodes = newNodes
	default:
		retUpstream.Nodes = make(v1beta1.BkGatewayNodeList, 0)
	}

	for idx := range retUpstream.Nodes {
		node := &retUpstream.Nodes[idx]
		node.Host = render.GetURIRender().Render(node.Host, c.stage.Spec.Vars)
		host, port, err := net.SplitHostPort(node.Host)
		if err == nil {
			porti, ierr := strconv.Atoi(port)
			node.Host = host
			if ierr != nil {
				c.logger.Error(err, "convert node hosts port to int failed", "host", host, "port", port)
			} else {
				node.Port = porti
			}
		}
	}

	if len(upstream.ServiceName) != 0 && (retUpstream.Nodes == nil || len(retUpstream.Nodes) == 0) {
		retUpstream.ServiceName = utils.StringPtr(upstream.ServiceName)
	}

	if upstream.TLSEnable {
		if upstream.PassHost == passHostPass {
			err := eris.New("upstream passHost should be 'node' or 'rewrite' when tls enabled")
			c.logger.Error(err, "", "type", typeMeta, "obj", objMeta)
			return retUpstream, err
		}

		if upstream.PassHost == passHostRewrite {
			if len(upstream.UpstreamHost) == 0 {
				err := eris.New("upstream upstreamHost should be set when passHost is rewrite")
				c.logger.Error(err, "", "type", typeMeta, "obj", objMeta)
				return retUpstream, err
			}
			prefix, tlsCert, found := c.upstreamConfig.CertDetectTree.MatchLongestPrefixWithRandomCert(
				upstream.UpstreamHost,
			)
			if !found {
				err := eris.New("No suitable cert for host")
				c.logger.Error(err, "", "host", upstream.PassHost, "type", typeMeta, "obj", objMeta)
				return retUpstream, err
			}
			c.logger.Debugw("Matched cert", "prefix", prefix, "cert", tlsCert, "type", typeMeta, "obj", objMeta)
			retUpstream.TLS = &apisix.UpstreamTLS{
				ClientCert: tlsCert.Cert,
				ClientKey:  tlsCert.Key,
			}
		}

		if upstream.PassHost == passHostNode {
			if len(retUpstream.Nodes) == 0 {
				err := eris.New("upstream nodes should be set when passHost is node")
				c.logger.Error(err, "", "type", typeMeta, "obj", objMeta)
				return retUpstream, err
			}
			certMatchCount := make(map[*v1beta1.TLSCert]int)
			for _, node := range retUpstream.Nodes {
				prefix, tlsCerts, found := c.upstreamConfig.CertDetectTree.MatchLongestPrefix(node.Host)
				if !found {
					err := eris.New("No suitable cert for host")
					c.logger.Error(err, "", "node", node, "type", typeMeta, "obj", objMeta)
					return retUpstream, err
				}
				c.logger.Debugw(
					"Matched cert",
					"node",
					node,
					"prefix",
					prefix,
					"certs",
					tlsCerts,
					"type",
					typeMeta,
					"obj",
					objMeta,
				)
				for _, cert := range tlsCerts {
					certMatchCount[cert]++
				}
			}

			for cert, count := range certMatchCount {
				if count == len(retUpstream.Nodes) {
					c.logger.Debugw(
						"Matched cert for all node",
						"nodes",
						retUpstream.Nodes,
						"certs",
						cert,
						"type",
						typeMeta,
						"obj",
						objMeta,
					)
					retUpstream.TLS = &apisix.UpstreamTLS{
						ClientCert: cert.Cert,
						ClientKey:  cert.Key,
					}
					return retUpstream, nil
				}
			}

			err := eris.New("No suitable cert for all nodes")
			c.logger.Error(err, "", "nodes", retUpstream.Nodes, "type", typeMeta, "obj", objMeta)
			return retUpstream, err
		}
	}
	return retUpstream, nil
}

func (c *Converter) convertKubernetesServiceNodes(ns, svcName string) ([]v1beta1.BkGatewayNode, error) {
	if c.upstreamConfig.NodeDiscoverer != nil {
		return c.upstreamConfig.NodeDiscoverer.GetNodes(ns, svcName)
	}
	return nil, eris.Errorf("kube service converter not register")
}

func (c *Converter) externalExternalServiceNodes(
	typeMeta metav1.TypeMeta,
	objMeta metav1.ObjectMeta,
	svcType string,
) ([]v1beta1.BkGatewayNode, error) {
	if c.upstreamConfig.ExternalNodeDiscoverer != nil {
		return c.upstreamConfig.ExternalNodeDiscoverer.GetNodes(
			frametypes.KindMapping[typeMeta.Kind],
			objMeta.Namespace,
			svcType,
			objMeta.Name,
			c.gatewayName,
			c.stageName,
		)
	}
	return nil, eris.Errorf("external service converter not register")
}

func (c *Converter) getProxyRewrite(
	rewrite *v1beta1.BkGatewayResourceHTTPRewrite,
	resource *v1beta1.BkGatewayResource,
) (map[string]interface{}, error) {
	if !rewrite.Enabled {
		return map[string]interface{}{}, nil
	}

	rewritePluginConfig := make(map[string]interface{})
	if len(rewrite.Path) != 0 {
		upstreamURI := render.GetUpstreamURIRender().Render(rewrite.Path, c.stage.Spec.Vars)
		if resource.Spec.MatchSubPath {
			rewritePluginConfig["uri"] = fmt.Sprintf(
				"%s/${%s}",
				strings.TrimSuffix(upstreamURI, "/"),
				config.BKAPIGatewaySubpathMatchParamName,
			)
			rewritePluginConfig["match_subpath"] = true
			rewritePluginConfig["subpath_param_name"] = config.BKAPIGatewaySubpathMatchParamName
		} else {
			rewritePluginConfig["uri"] = upstreamURI
		}
	}

	if len(rewrite.Method) != 0 && rewrite.Method != "ANY" {
		rewritePluginConfig["method"] = rewrite.Method
	}

	return map[string]interface{}{
		pluginNameBKProxyRewrite: rewritePluginConfig,
	}, nil
}

func (c *Converter) appendStagePlugins(stagePlugins map[string]interface{}) {
	if len(c.stage.Spec.Plugins) == 0 {
		return
	}
	for _, plugin := range c.stage.Spec.Plugins {
		if plugin == nil {
			continue
		}
		pname, pconfig := c.convertPlugin(plugin)
		if _, ok := stagePlugins[pname]; ok {
			continue
		}
		stagePlugins[pname] = pconfig
	}
}

func (c *Converter) convertHTTPTimeout(timeout *v1beta1.UpstreamTimeout) *apisixv1.UpstreamTimeout {
	if timeout == nil {
		return nil
	}
	retTimeout := &apisixv1.UpstreamTimeout{
		Connect: int(v1beta1.ParseDuration(timeout.Connect).Seconds()),
		Read:    int(v1beta1.ParseDuration(timeout.Read).Seconds()),
		Send:    int(v1beta1.ParseDuration(timeout.Send).Seconds()),
	}
	if retTimeout.Connect <= 0 {
		retTimeout.Connect = 60
	}
	if retTimeout.Read <= 0 {
		retTimeout.Read = 60
	}
	if retTimeout.Send <= 0 {
		retTimeout.Send = 60
	}
	return retTimeout
}
