debug: true

operator:
  defaultGateway: "bk-default"
  defaultStage: "default"
  #write apisix etcd interval
  etcdPutInterval: "100ms"
  etcdDelInterval: "15s"

dashboard:
  etcd:
    endpoints: "bk-apigateway-etcd:2379"
    keyPrefix: "/bk-gateway-apigw/default"
    username: "root"
    password: "blueking"

apisix:
  etcd:
    endpoints: "bk-apigateway-etcd:2379"
    keyPrefix: "/bk-gateway-apisix"
    username: "root"
    password: "blueking"
  resourceStoreMode: "etcd"
  # 内置插件列表，这些插件无需进行schema校验（按优先级排序）
  innerPlugins:
    # priority: 18880
    - "bk-legacy-invalid-params"
    # priority: 18870 (will be deprecated)
    - "bk-opentelemetry"
    # priority: 18860
    - "bk-not-found-handler"
    # priority: 18850
    - "bk-request-id"
    # priority: 18840
    - "bk-stage-context"
    # priority: 18825
    - "bk-backend-context"
    # priority: 18820
    - "bk-resource-context"
    # priority: 18815 (will be deprecated)
    - "bk-status-rewrite"
    # priority: 18810 (will be deprecated)
    - "bk-verified-user-exempted-apps"
    # priority: 18809
    - "bk-real-ip"
    # priority: 18800
    - "bk-log-context"
    # priority: 18735
    - "bk-access-token-source"
    # priority: 18730
    - "bk-auth-verify"
    # priority: 18725
    - "bk-username-required"
    # priority: 17900
    - "bk-cors"
    # priority: 17700
    - "bk-break-recursive-call"
    # priority: 17690
    - "bk-request-body-limit"
    # priority: 17680
    - "bk-auth-validate"
    # priority: 17679
    - "bk-user-restriction"
    # priority: 17675
    - "bk-tenant-verify"
    # priority: 17674
    - "bk-tenant-validate"
    # priority: 17670
    - "bk-jwt"
    # priority: 17662
    - "bk-ip-restriction"
    # priority: 17660 (disabled by default)
    - "bk-concurrency-limit"
    # priority: 17653
    - "bk-resource-rate-limit"
    # priority: 17652
    - "bk-stage-rate-limit"
    # priority: 17651 (not used, but just keep in codebase)
    - "bk-stage-global-rate-limit"
    # priority: 17640
    - "bk-permission"
    # priority: 17460
    - "bk-traffic-label"
    # priority: 17450
    - "bk-delete-sensitive"
    # priority: 17440
    - "bk-delete-cookie"
    # priority: 17430
    - "bk-proxy-rewrite"
    # priority: 17425
    - "bk-default-tenant"
    # priority: 17421
    - "bk-stage-header-rewrite"
    # priority: 17420
    - "bk-resource-header-rewrite"
    # priority: 17150
    - "bk-mock"
    # priority: 153
    - "bk-response-check"
    # priority: 145
    - "bk-debug"
    # priority: 0
    - "bk-error-wrapper"
    - "bk-repl-debugger"
    # 其他内置插件
    - "prometheus"
  virtualStage:
    extraApisixResources: "/data/config/extra-resources.yaml"

eventReporter:
  coreAPIHost: "bk-apigateway-core-api:80"
  apisixHost: "bk-apigateway-apigateway"
  versionProbe:
    timout: "2m" # version probe timeout
    waitTime: "15s" # version probe wait time
    bufferSize: 300 # version probe chain size
    retry:
      count: 60
      interval: "500ms"
  eventBufferSize: 300 # reporter eventChain size
  reporterBufferSize: 100 # control currency fo report to core API

auth:
  # should configured same as the apisix:conf/config.yaml bk_gateway.instance.{id, secret}
  id: "faf44a48-59e9-f790-2412-e56c90551fb3"
  secret: "358627d8-d3e8-4522-8f16-b5530776bbb8"

httpServer:
  bindAddress: "0.0.0.0"
  bindAddressV6: "[::]"
  bindPort: 6004
# The authentication pwd used to access the API
  authPassword: DebugModel@bk

logger:
  default:
    level: info
    writer: os
    settings: {name: stdout}
  controller:
    level: info
    writer: os
    settings: {name: stdout}
    # writer: file

sentry:
  dsn: ""
  ## zapcore.Level
  reportLevel: 3

tracing:
  enable: false
  endpoint: "127.0.0.1:4318"
  ## report type: grpc/http
  type: "http"
  ## support: "always_on"/"always_off"/"trace_id_ratio"/"parentbased_always_on",if not config,default: "trace_id_ratio"
  sampler: "trace_id_ratio"
  samplerRatio: 0.001
  token: "blueking"
  serviceName: "blueking-apigateway-operator"