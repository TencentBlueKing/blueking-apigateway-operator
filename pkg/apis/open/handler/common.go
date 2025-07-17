package handler

import (
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// ResourceHandler resource api handler
type ResourceHandler struct {
	LeaderElector   leaderelection.LeaderElector
	registry        registry.Registry
	committer       *commiter.Commiter
	apisixConfStore synchronizer.ApisixConfigStore
}

// NewResourceApi constructor of resource handler
func NewResourceApi(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore,
) *ResourceHandler {
	return &ResourceHandler{
		LeaderElector:   leaderElector,
		registry:        registry,
		committer:       committer,
		apisixConfStore: apiSixConfStore,
	}
}
