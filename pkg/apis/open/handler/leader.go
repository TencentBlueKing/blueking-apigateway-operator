package handler

import (
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetLeader get leader pod host
func (r *ResourceHandler) GetLeader(c *gin.Context) {
	if r.LeaderElector == nil {
		utils.BaseErrorJSONResponse(c, utils.NotFoundError, "LeaderElector not found", http.StatusOK)
		return
	}
	utils.SuccessJSONResponse(c, r.LeaderElector.Leader())
}
