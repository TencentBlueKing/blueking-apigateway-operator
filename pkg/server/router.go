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

package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/api"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

// NewRouter do the router initialization
func NewRouter(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore,
	router *gin.Engine,
	conf *config.Config,
) *gin.Engine {
	router.GET("/ping", func(c *gin.Context) {
		utils.SuccessJSONResponse(c, "ok")
	})
	router.GET("/healthz", func(c *gin.Context) {
		utils.SuccessJSONResponse(c, "ok")
	})
	operatorRouter := router.Group("/v1")
	operatorRouter.Use(gin.BasicAuth(gin.Accounts{
		constant.ApiAuthAccount: conf.HttpServer.AuthPassword,
	}))
	operatorRouter.Use(gin.Recovery())
	api.Register(operatorRouter, leaderElector, registry, committer, apiSixConfStore)

	// pprof
	pprofRouter := router.Group("/debug/pprof")
	if !conf.Debug {
		pprofRouter.Use(gin.BasicAuth(gin.Accounts{
			"bk-apigateway": "DebugModel@bk",
		}))
	}
	{
		pprofRouter.GET("/", pprofHandler(pprof.Index))
		pprofRouter.GET("/cmdline", pprofHandler(pprof.Cmdline))
		pprofRouter.GET("/profile", pprofHandler(pprof.Profile))
		pprofRouter.POST("/symbol", pprofHandler(pprof.Symbol))
		pprofRouter.GET("/symbol", pprofHandler(pprof.Symbol))
		pprofRouter.GET("/trace", pprofHandler(pprof.Trace))
		pprofRouter.GET("/allocs", pprofHandler(pprof.Handler("allocs").ServeHTTP))
		pprofRouter.GET("/block", pprofHandler(pprof.Handler("block").ServeHTTP))
		pprofRouter.GET("/goroutine", pprofHandler(pprof.Handler("goroutine").ServeHTTP))
		pprofRouter.GET("/heap", pprofHandler(pprof.Handler("heap").ServeHTTP))
		pprofRouter.GET("/mutex", pprofHandler(pprof.Handler("mutex").ServeHTTP))
		pprofRouter.GET("/threadcreate", pprofHandler(pprof.Handler("threadcreate").ServeHTTP))
	}
	return router
}

func pprofHandler(h http.HandlerFunc) gin.HandlerFunc {
	// handler := http.HandlerFunc(h)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
