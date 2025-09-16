debug: true

operator:
  withKube: true
  withLeader: true
  agentMode: true

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

instance:
  id: "coreapi"
  secret: "coreapi"

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