## debug cli tool
Provide functions such as comparison, manual synchronization, and viewing of apisix resources on the data plane for the configuration data of the control plane and the gateway configuration data of the data plane
## Features

```shell
micro-gateway-operator --help
```
output：
```shell
bk-gateway operator for apisix

Usage:
  micro-gateway-operator [flags]
  micro-gateway-operator [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  diff        diff between bkgateway resources and apisix storage
  help        Help about any command
  list        list resources in apisix
  sync        sync bkgateway resources into apisix storage

Flags:
  -c, --config string   config file (default is config.yml;required)
  -h, --help            help for micro-gateway-operator
  -v, --version         version for micro-gateway-operator
      --viper           Use Viper for configuration (default true)
```
### diff
It mainly provides the comparison function of control plane resources and data plane gateway configuration resources

```shell
diff between bkgateway resources and apisix storage

Usage:
  micro-gateway-operator diff [flags]

Flags:
      --all                    list all gateway resources
  -c, --config string          config file (default is config.yml;required)
      --gateway string         gateway name for list command
  -h, --help                   help for diff
      --resource_id int        resource ID for list command, default(-1) for all resources in stage (default -1)
      --resource_name string   resource Name for list command, empty for all resources in stage
      --stage string           stage name for list command
      --viper                  Use Viper for configuration (default true)
  -w, --write-out string       response write out format (simple, json, yaml) (default "simple")
```

### list
Provide data plane gateway resource function query
```shell
list resources in apisix

Usage:
  micro-gateway-operator list [flags]

Flags:
      --all                    list all gateway resources
  -l, --config string          config file (default is config.yml;required)
      --gateway string         gateway name for list command
  -h, --help                   help for list
      --resource_id int        resource ID for list command, default(-1) for all resources in stage (default -1)
      --resource_name string   resource name for list command, empty for all resources in stage. Can not be set with resource_id simultaneously
      --stage string           stage name for list command
      --viper                  Use Viper for configuration (default true)
  -w, --write-out string       response write out format (simple, json, yaml) (default "json")
```
### sync
Manually synchronize control plane data to data plane gateway resource data
```shell
sync bkgateway resources into apisix storage

Usage:
  micro-gateway-operator sync [flags]

Flags:
      --all              sync all gateway
  -c, --config string    config file (default is config.yml;required)
      --gateway string   gateway for sync command
  -h, --help             help for sync
      --stage string     stage for sync command
      --viper            Use Viper for configuration (default true)
```