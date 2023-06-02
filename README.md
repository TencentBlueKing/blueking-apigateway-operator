![img](https://github.com/TencentBlueKing/blueking-apigateway/raw/master/docs/resource/img/blueking_apigateway_zh.png)
---

[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat)](https://github.com/TencentBlueKing/blueking-apigateway-operator/blob/main/LICENSE.txt) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/TencentBlueKing/blueking-apigateway-operator/pulls)

简体中文 | [English](README_EN.md)

## 概览

蓝鲸 API 网关（API Gateway），是一种高性能、高可用的 API 托管服务，可以帮助开发者创建、发布、维护、监控和保护 API， 以快速、低成本、低风险地对外开放蓝鲸应用或其他系统的数据或服务

本项目是 `蓝鲸 API 网关 - Operator`。

**蓝鲸 API 网关核心服务开源项目**

- 蓝鲸 API 网关 - [控制面](https://github.com/TencentBlueKing/blueking-apigateway)
- 蓝鲸 API 网关 - [数据面](https://github.com/TencentBlueKing/blueking-apigateway-apisix)
- 蓝鲸 API 网关 - [Operator](https://github.com/TencentBlueKing/blueking-apigateway-operator)

## 功能特性

- 转换控制面资源为数据面资源：Operator 负责对接控制面（Control Plane）和数据面（Data Plane），可以将从控制面下发的配置资源转换为数据面的网关描述配置资源。
- 支持多种资源来源：Operator 支持 etcd 和 Kubernetes 云原生的方式来处理资源数据，以用于管理和协调控制面和数据面数据流转。
- 支持专项网关和共享网关：Operator 面向不同的使用场景，可以将网关作为一个组件集成到其它系统中，也可以将其作为公共网关组件来使用。
- 提供debug cli工具：为控制面配置数据和数据面的网关配置数据提供对比、手动同步以及查看数据面apisix资源等功能。详细介绍:[debug cli工具文档](./docs/debug/README.md)
## 快速开始

- [本地开发部署指引](https://github.com/TencentBlueKing/blueking-apigateway/blob/master/docs/DEVELOP_GUIDE.md)

## 支持

- [蓝鲸 API 网关产品白皮书](https://bk.tencent.com/docs/document/7.0/171/13974)
- [蓝鲸智云 - 学习社区](https://bk.tencent.com/s-mart/community)
- [蓝鲸 DevOps 在线视频教程](https://bk.tencent.com/s-mart/video)
- 加入技术交流 QQ 群：

![img](https://github.com/TencentBlueKing/blueking-apigateway/raw/master/docs/resource/img/bk_qq_group.png)

## 蓝鲸社区

- [BK-CI](https://github.com/TencentBlueKing/bk-ci)：蓝鲸持续集成平台是一个开源的持续集成和持续交付系统，可以轻松将你的研发流程呈现到你面前。
- [BK-BCS](https://github.com/TencentBlueKing/bk-bcs)：蓝鲸容器管理平台是以容器技术为基础，为微服务业务提供编排管理的基础服务平台。
- [BK-SOPS](https://github.com/TencentBlueKing/bk-sops)：标准运维（SOPS）是通过可视化的图形界面进行任务流程编排和执行的系统，是蓝鲸体系中一款轻量级的调度编排类
  SaaS 产品。
- [BK-CMDB](https://github.com/TencentBlueKing/bk-cmdb)：蓝鲸配置平台是一个面向资产及应用的企业级配置管理平台。
- [BK-JOB](https://github.com/TencentBlueKing/bk-job)：蓝鲸作业平台（Job）是一套运维脚本管理系统，具备海量任务并发处理能力。

## 贡献

如果你有好的意见或建议，欢迎给我们提 Issues 或 PullRequests，为蓝鲸开源社区贡献力量。关于分支 / Issue 及 PR,
请查看 [CONTRIBUTING](https://github.com/TencentBlueKing/blueking-apigateway/blob/master/docs/CONTRIBUTING.md)。

[腾讯开源激励计划](https://opensource.tencent.com/contribution) 鼓励开发者的参与和贡献，期待你的加入。
