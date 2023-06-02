![img](https://github.com/TencentBlueKing/blueking-apigateway/raw/master/docs/resource/img/blueking_apigateway_en.png)
---

[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](https://github.com/TencentBlueKing/blueking-apigateway-operator/blob/main/LICENSE.txt) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/TencentBlueKing/blueking-apigateway-operator/pulls)

[简体中文](README.md) | English

## Overview

BlueKing API Gateway is a high-performance and highly available API hosting service that helps developers create, publish, maintain, monitor, and protect APIs, allowing for quick, low-cost, and low-risk access to data or services from BlueKing applications or other systems.

This project is the "BlueKing API Gateway - Operator".

**BlueKing API Gateway Core Services Open Source Projects**

- BlueKing API Gateway - [Control Plane](https://github.com/TencentBlueKing/blueking-apigateway)
- BlueKing API Gateway - [Data Plane](https://github.com/TencentBlueKing/blueking-apigateway-apisix)
- BlueKing API Gateway - [Operator](https://github.com/TencentBlueKing/blueking-apigateway-operator)

## Features

- Convert Control Plane resources to Data Plane resources: The Operator is responsible for integrating with the Control Plane and Data Plane and can convert configuration resources issued from the Control Plane into gateway descriptor configuration resources for the Data Plane.
- Support for multiple resource sources: The Operator supports etcd and Kubernetes native cloud methods for processing resource data to manage and coordinate data flow between the Control Plane and Data Plane.
- Support for specialized and shared gateways: The Operator is designed for different usage scenarios and can integrate with gateways as a component in other systems or used as a common gateway component.
- Provide debug cli tools: provide functions such as comparison, manual synchronization, and viewing of apisix resources on the data plane for the configuration data of the control plane and the gateway configuration data of the data plane. detailed introduction:[debug cli tool documentation](./docs/debug/README_EN.md)
## Getting started

- [Local Developing(In Chinese)](https://github.com/TencentBlueKing/blueking-apigateway/blob/master/docs/DEVELOP_GUIDE.md)

## Support

- [white paper(In Chinese)](https://bk.tencent.com/docs/document/7.0/171/13974)
- [bk forum](https://bk.tencent.com/s-mart/community)
- [bk DevOps online video tutorial(In Chinese)](https://bk.tencent.com/s-mart/video)
- Join technical exchange QQ group:

![img](https://github.com/TencentBlueKing/blueking-apigateway/raw/master/docs/resource/img/bk_qq_group.png)

## BlueKing Community

- [BK-CI](https://github.com/TencentBlueKing/bk-ci): a continuous integration and continuous delivery system that can
  easily present your R & D process to you.
- [BK-BCS](https://github.com/TencentBlueKing/bk-bcs): a basic container service platform which provides orchestration
  and management for micro-service business.
- [BK-SOPS](https://github.com/TencentBlueKing/bk-sops): an lightweight scheduling SaaS for task flow scheduling and
  execution through a visual graphical interface.
- [BK-CMDB](https://github.com/TencentBlueKing/bk-cmdb): an enterprise-level configuration management platform for
  assets and applications.
- [BK-JOB](https://github.com/TencentBlueKing/bk-job): BlueKing JOB is a set of operation and maintenance script
  management platform with the ability to handle a large number of tasks concurrently.

## Contributing

If you have good ideas or suggestions, please let us know by Issues or Pull Requests and contribute to the Blue Whale
Open Source Community. For blueking-apigateway branch management, issues, and pr specifications, read
the [CONTRIBUTING(In Chinese)](https://github.com/TencentBlueKing/blueking-apigateway/blob/master/docs/CONTRIBUTING.md)

If you are interested in contributing, check out the [CONTRIBUTING.md], also join
our [Tencent OpenSource Plan](https://opensource.tencent.com/contribution).

## License

blueking-apigateway is based on the MIT protocol. Please refer to [LICENSE](LICENSE.txt)