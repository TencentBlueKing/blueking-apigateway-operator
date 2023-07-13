debug: true

operator:
  withKube: true
  withLeader: true
  agentMode: true

  defaultGateway: "bk-default"
  defaultStage: "default"

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
    operatorExternalHost: "bk-apigateway-operator"
    operatorExternalHealthProbePort: 6004
    extraApisixResources: "/data/config/extra-resources.yaml"

eventReporter:
  coreAPIHost: "bk-apigateway-core-api:80"
  apisixHost: "bk-apigateway-apigateway"
  versionProbe:
    timout: "2m" # version probe timeout
    bufferSize: 300 # version probe chain size
    retry:
      count: 60
      interval: "500ms"
  eventBufferSize: 300 # reporter eventChain size
  reporterBufferSize: 100 # control currency fo report to core API


instance:
  ID:"coreapi"
  Secret:"coreapi"



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
