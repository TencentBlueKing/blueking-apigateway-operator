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

// Package config provides the configuration for the BlueKing API Gateway Operator.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

// BKAPIGatewayLabelKeyGatewayName ...
const (
	BKAPIGatewayLabelKeyGatewayName      = "gateway.bk.tencent.com/gateway"
	BKAPIGatewayLabelKeyGatewayStage     = "gateway.bk.tencent.com/stage"
	BKAPIGatewayLabelKeyGatewayPublishID = "gateway.bk.tencent.com/publish_id"
	BKAPIGatewayLabelKeyResourceName     = "gateway.bk.tencent.com/name"

	BKAPIGatewaySubpathMatchParamName = "bk_api_subpath_match_param_name"

	SecretCACertKey = "ca.crt"
	SecretCertKey   = "tls.crt"
	SecretKeyKey    = "tls.key"
)

const (
	envPodName      = "BK_GATEWAY_POD_NAME"
	envPodNamespace = "BK_GATEWAY_POD_NAMESPACE"
	envPodIP        = "BK_GATEWAY_POD_IP"
)

// InstanceName ...
var (
	InstanceName      string
	InstanceNamespace string
	InstanceIP        string

	DefaultStageKey string
	VirtualStageKey string
)

// Metric ...
type Metric struct {
	BindAddress   string
	BindAddressV6 string
	BindPort      int
}

// HttpServer ...
type HttpServer struct {
	BindAddress   string
	BindAddressV6 string
	BindPort      int
	AuthPassword  string // The authentication pwd used to access the API
}

// VirtualStage ...
type VirtualStage struct {
	VirtualGateway string
	VirtualStage   string

	ExtraApisixResources string
	FileLoggerLogPath    string
}

// Dashboard ...
type Dashboard struct {
	Etcd Etcd

	PrefixPrepend      bool
	ResourceBasePrefix string
}

// Apisix ...
type Apisix struct {
	Etcd Etcd

	ResourceStoreMode     string
	ConfigPath            string
	InternalDiscoveryType []string

	VirtualStage VirtualStage
}

// Operator ...
type Operator struct {
	WithKube   bool
	WithLeader bool
	AgentMode  bool

	DefaultGateway string
	DefaultStage   string

	AgentEventsWaitingTimeWindow time.Duration
	AgentForceUpdateTimeWindow   time.Duration
	AgentCommitTimeWindow        time.Duration
	AgentConcurrencyLimit        int

	// etcd put interval
	EtcdPutInterval time.Duration
}

// VersionProbe ...
type VersionProbe struct {
	BufferSize int
	Retry      Retry
	Timeout    time.Duration
	WaitTime   time.Duration
}

// Retry ...
type Retry struct {
	Count    int
	Interval time.Duration
}

// Instance ...
type Instance struct {
	ID     string
	Secret string
}

// EventReporter ...
type EventReporter struct {
	CoreAPIHost        string
	ApisixHost         string
	VersionProbe       VersionProbe
	EventBufferSize    int
	ReporterBufferSize int
}

// KubeExtension ...
type KubeExtension struct {
	WorkNamespace string

	LeaderElectionType          string
	LeaderElectionName          string
	LeaderElectionLeaseDuration int
	LeaderElectionRenewDuration int
	LeaderElectionRetryDuration int
}

// Etcd ...
type Etcd struct {
	Endpoints   string
	CACert      string
	Cert        string
	Key         string
	Username    string
	Password    string
	WithoutAuth bool

	KeyPrefix string
}

// Sentry ...
type Sentry struct {
	Dsn         string
	ReportLevel int
}

// Logger ...
type Logger struct {
	Default    LogConfig
	Controller LogConfig
}

// LogConfig ...
type LogConfig struct {
	Level    string
	Writer   string
	Settings map[string]string
}

// Tracing ...
type Tracing struct {
	Enable       bool
	Endpoint     string
	Type         string
	Token        string
	Sampler      string
	SamplerRatio float64
	ServiceName  string
}

// Config ...
type Config struct {
	Debug bool

	HttpServer HttpServer

	Dashboard     Dashboard
	Apisix        Apisix
	Operator      Operator
	EventReporter EventReporter
	Instance      Instance

	KubeExtension KubeExtension

	Logger  Logger
	Sentry  Sentry
	Tracing Tracing
}

func newDefaultConfig() *Config {
	return &Config{
		Debug: false,
		HttpServer: HttpServer{
			BindPort:     6004,
			AuthPassword: "DebugModel@bk",
		},
		KubeExtension: KubeExtension{
			LeaderElectionType:          "leases",
			LeaderElectionName:          "election.gateway.bk.tencent.com",
			LeaderElectionLeaseDuration: 30,
			LeaderElectionRenewDuration: 25,
			LeaderElectionRetryDuration: 5, // retry_count= Ceil(RenewDeadline / RetryPeriod)
		},
		Dashboard: Dashboard{
			Etcd: Etcd{
				KeyPrefix: "/bk-gateway/default",
			},
			ResourceBasePrefix: "/api",
		},
		Apisix: Apisix{
			Etcd: Etcd{
				KeyPrefix: "/apisix",
			},
			ResourceStoreMode:     "etcd",
			ConfigPath:            "/usr/local/apisix/conf/apisix.yaml",
			InternalDiscoveryType: []string{"dns", "consul_kv", "nacos", "eureka"},
			VirtualStage: VirtualStage{
				FileLoggerLogPath: "/usr/local/apisix/logs/access.log",

				VirtualGateway: "-",
				VirtualStage:   "-",
			},
		},
		EventReporter: EventReporter{
			VersionProbe: VersionProbe{
				BufferSize: 100,
				Retry: Retry{
					Count:    60,
					Interval: time.Second,
				},
				Timeout:  time.Minute * 2,
				WaitTime: time.Second * 15,
			},
			EventBufferSize:    300,
			ReporterBufferSize: 100,
		},
		Operator: Operator{
			DefaultGateway: "-",
			DefaultStage:   "global",

			AgentEventsWaitingTimeWindow: 2 * time.Second,
			AgentForceUpdateTimeWindow:   10 * time.Second,
			AgentCommitTimeWindow:        5 * time.Second,
			AgentConcurrencyLimit:        2,
			EtcdPutInterval:              50 * time.Millisecond,
		},
		Sentry: Sentry{
			ReportLevel: 2,
		},
		Tracing: Tracing{},
		Logger: Logger{
			Default: LogConfig{
				Level:  "info",
				Writer: "os",
				Settings: map[string]string{
					"name": "stdout",
				},
			},
			Controller: LogConfig{
				Level:  "info",
				Writer: "os",
				Settings: map[string]string{
					"name": "stdout",
				},
			},
		},
	}
}

// Load ...
func Load(v *viper.Viper) (*Config, error) {
	cfg := newDefaultConfig()
	// 将配置信息绑定到结构体上
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	cfg.init()

	return cfg, nil
}

func (c *Config) init() {
	instanceName := os.Getenv(envPodName)
	instanceNamespace := os.Getenv(envPodNamespace)
	instanceIP := os.Getenv(envPodIP)

	if c.Operator.WithKube {
		if len(instanceName) == 0 || len(instanceNamespace) == 0 {
			fmt.Printf("%v or %v is empty", envPodName, envPodNamespace)
			os.Exit(1)
		}

		InstanceName = instanceName
		InstanceNamespace = instanceNamespace
		InstanceIP = instanceIP
	} else {
		hname, _ := os.Hostname()
		InstanceName = hname + "_" + utils.GetGeneratedUUID()
	}

	DefaultStageKey = GenStagePrimaryKey(c.Operator.DefaultGateway, c.Operator.DefaultStage)
	VirtualStageKey = GenStagePrimaryKey(c.Apisix.VirtualStage.VirtualGateway, c.Apisix.VirtualStage.VirtualStage)

	c.Apisix.Etcd.KeyPrefix = strings.TrimSuffix(c.Apisix.Etcd.KeyPrefix, "/")
	c.Dashboard.Etcd.KeyPrefix = strings.TrimSuffix(c.Dashboard.Etcd.KeyPrefix, "/")

	if c.Debug {
		by, _ := json.Marshal(c)
		fmt.Println(string(by))
	}
}

// GenStagePrimaryKey build apisix configuration stage key from gateway name and stage name
func GenStagePrimaryKey(gatewayName string, stageName string) string {
	if len(gatewayName) == 0 && len(stageName) == 0 {
		return DefaultStageKey
	}
	return fmt.Sprintf("%s/%s", gatewayName, stageName)
}
