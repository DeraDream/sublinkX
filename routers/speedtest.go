package routers

import (
	"sublink/api"

	"github.com/gin-gonic/gin"
)

func SpeedTest(r *gin.Engine) {
	admin := r.Group("/api/v1/speedtest")
	{
		admin.POST("/agents/create", api.CreateHomeAgent)
		admin.GET("/agents", api.ListHomeAgents)
		admin.POST("/agents/mode", api.SetHomeAgentMode)
		admin.DELETE("/agents", api.DeleteHomeAgent)
		admin.POST("/tasks", api.CreateSpeedTestTask)
		admin.GET("/tasks", api.ListSpeedTestTasks)
	}

	agent := r.Group("/api/v1/agent")
	{
		agent.POST("/poll", api.HomeAgentPoll)
		agent.POST("/report", api.HomeAgentReport)
	}
}
