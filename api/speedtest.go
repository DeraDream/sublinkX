package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sublink/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type agentTaskResult struct {
	TaskID       uint    `json:"task_id"`
	Success      bool    `json:"success"`
	LatencyMs    int64   `json:"latency_ms"`
	DownloadMbps float64 `json:"download_mbps"`
	TestBytes    int64   `json:"test_bytes"`
	EgressIP     string  `json:"egress_ip"`
	Error        string  `json:"error"`
}

func randomAgentToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func publicBaseURL(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}

func installCommand(baseURL, token string) string {
	return fmt.Sprintf(
		`sudo env http_proxy="${http_proxy:-}" https_proxy="${https_proxy:-}" HTTP_PROXY="${HTTP_PROXY:-}" HTTPS_PROXY="${HTTPS_PROXY:-}" ALL_PROXY="${ALL_PROXY:-}" curl -fL --retry 3 --connect-timeout 15 "https://github.com/DeraDream/sublinkX/releases/latest/download/sublink_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o /var/tmp/sublink-agent.new && sudo chmod 755 /var/tmp/sublink-agent.new && sudo /var/tmp/sublink-agent.new agent install --server %s --token %s && sudo rm -f /var/tmp/sublink-agent.new`,
		baseURL,
		token,
	)
}

func upgradeCommand() string {
	return `sudo env http_proxy="${http_proxy:-}" https_proxy="${https_proxy:-}" HTTP_PROXY="${HTTP_PROXY:-}" HTTPS_PROXY="${HTTPS_PROXY:-}" ALL_PROXY="${ALL_PROXY:-}" curl -fL --retry 3 --connect-timeout 15 "https://github.com/DeraDream/sublinkX/releases/latest/download/sublink_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o /var/tmp/sublink-agent.new && sudo chmod 755 /var/tmp/sublink-agent.new && sudo /var/tmp/sublink-agent.new -version && sudo systemctl stop sublink-agent && sudo mv /var/tmp/sublink-agent.new /usr/local/bin/sublink-agent && sudo /usr/local/bin/sublink-agent agent install --config /etc/sublink-agent/config.yaml && sudo systemctl status sublink-agent --no-pager`
}

func CreateHomeAgent(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "名称不能为空"})
		return
	}
	token, err := randomAgentToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "生成令牌失败"})
		return
	}
	agent := models.HomeAgent{Name: name, TokenHash: models.HashAgentToken(token)}
	if err := models.DB.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": "00000",
		"data": gin.H{
			"agent":           agent,
			"token":           token,
			"install_command": installCommand(publicBaseURL(c), token),
		},
		"msg": "创建成功",
	})
}

func ListHomeAgents(c *gin.Context) {
	var agents []models.HomeAgent
	if err := models.DB.Order("id desc").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	type agentView struct {
		models.HomeAgent
		Online         bool   `json:"online"`
		State          string `json:"state"`
		Pending        int64  `json:"pending_tasks"`
		UpgradeCommand string `json:"upgrade_command"`
	}
	out := make([]agentView, 0, len(agents))
	now := time.Now()
	for _, agent := range agents {
		var pending int64
		models.DB.Model(&models.SpeedTestTask{}).
			Where("home_agent_id = ? AND status IN ?", agent.ID, []string{models.SpeedTaskPending, models.SpeedTaskRunning}).
			Count(&pending)
		online := agent.LastSeen != nil && now.Sub(*agent.LastSeen) < 2*time.Minute
		state := "suspended"
		if agent.PersistentActive || pending > 0 {
			state = "active"
		}
		out = append(out, agentView{
			HomeAgent: agent, Online: online, State: state, Pending: pending,
			UpgradeCommand: upgradeCommand(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "data": out, "msg": "获取成功"})
}

func SetHomeAgentMode(c *gin.Context) {
	id, _ := strconv.Atoi(c.PostForm("id"))
	active := c.PostForm("active") == "true"
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "id 不正确"})
		return
	}
	if err := models.DB.Model(&models.HomeAgent{}).Where("id = ?", id).
		Update("persistent_active", active).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "msg": "状态已更新"})
}

func DeleteHomeAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "id 不正确"})
		return
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("home_agent_id = ?", id).Delete(&models.SpeedTestTask{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.HomeAgent{}, id).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "msg": "删除成功"})
}

func CreateSpeedTestTask(c *gin.Context) {
	nodeID, _ := strconv.Atoi(c.PostForm("node_id"))
	agentID, _ := strconv.Atoi(c.PostForm("agent_id"))
	testType := c.PostForm("type")
	if nodeID <= 0 || agentID <= 0 || (testType != "latency" && testType != "speed") {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "测速参数不正确"})
		return
	}
	var nd models.Node
	if err := models.DB.First(&nd, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "节点不存在"})
		return
	}
	var agent models.HomeAgent
	if err := models.DB.First(&agent, agentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "家宽测速端不存在"})
		return
	}
	if agent.LastSeen == nil || time.Since(*agent.LastSeen) >= 2*time.Minute {
		c.JSON(http.StatusConflict, gin.H{"msg": "家宽测速端当前离线"})
		return
	}
	scheme := strings.ToLower(strings.SplitN(nd.Link, "://", 2)[0])
	if scheme != "ss" && scheme != "vless" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "家宽测速目前仅支持 SS 和 VLESS 节点"})
		return
	}
	task := models.SpeedTestTask{
		HomeAgentID: agent.ID,
		NodeID:      nd.ID,
		NodeName:    nd.Name,
		TestType:    testType,
		Status:      models.SpeedTaskPending,
		NodeLink:    nd.Link,
	}
	if err := models.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "data": task, "msg": "测速任务已创建"})
}

func ListSpeedTestTasks(c *gin.Context) {
	var tasks []models.SpeedTestTask
	query := models.DB.Order("id desc").Limit(200)
	if taskID, _ := strconv.Atoi(c.Query("id")); taskID > 0 {
		query = query.Where("id = ?", taskID)
	}
	if nodeID, _ := strconv.Atoi(c.Query("node_id")); nodeID > 0 {
		query = query.Where("node_id = ?", nodeID)
	}
	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "data": tasks, "msg": "获取成功"})
}

func authenticateHomeAgent(c *gin.Context) (*models.HomeAgent, bool) {
	token := c.GetHeader("X-Agent-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "缺少测速端令牌"})
		return nil, false
	}
	var agent models.HomeAgent
	if err := models.DB.Where("token_hash = ?", models.HashAgentToken(token)).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "测速端令牌无效"})
		return nil, false
	}
	now := time.Now()
	updates := map[string]any{"last_seen": &now}
	if version := c.GetHeader("X-Agent-Version"); version != "" {
		updates["agent_version"] = version
	}
	if platform := c.GetHeader("X-Agent-Platform"); platform != "" {
		updates["platform"] = platform
	}
	models.DB.Model(&agent).Updates(updates)
	agent.LastSeen = &now
	return &agent, true
}

func HomeAgentPoll(c *gin.Context) {
	agent, ok := authenticateHomeAgent(c)
	if !ok {
		return
	}
	staleBefore := time.Now().Add(-5 * time.Minute)
	models.DB.Model(&models.SpeedTestTask{}).
		Where("home_agent_id = ? AND status = ? AND started_at < ?", agent.ID, models.SpeedTaskRunning, staleBefore).
		Updates(map[string]any{"status": models.SpeedTaskPending, "started_at": nil})

	var task models.SpeedTestTask
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("home_agent_id = ? AND status = ?", agent.ID, models.SpeedTaskPending).
			Order("id asc").First(&task)
		if result.Error != nil {
			return result.Error
		}
		now := time.Now()
		return tx.Model(&task).Updates(map[string]any{
			"status":     models.SpeedTaskRunning,
			"started_at": &now,
		}).Error
	})
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	mode := "suspended"
	pollAfter := 15
	if agent.PersistentActive || task.ID > 0 {
		mode = "active"
		pollAfter = 3
	}
	var taskData any
	if task.ID > 0 {
		taskData = gin.H{
			"id":        task.ID,
			"type":      task.TestType,
			"node_link": task.NodeLink,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": "00000",
		"data": gin.H{"mode": mode, "poll_after": pollAfter, "task": taskData},
		"msg":  "ok",
	})
}

func HomeAgentReport(c *gin.Context) {
	agent, ok := authenticateHomeAgent(c)
	if !ok {
		return
	}
	var result agentTaskResult
	if err := c.ShouldBindJSON(&result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "结果格式不正确"})
		return
	}
	var task models.SpeedTestTask
	if err := models.DB.Where("id = ? AND home_agent_id = ?", result.TaskID, agent.ID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "任务不存在"})
		return
	}
	now := time.Now()
	status := models.SpeedTaskSuccess
	if !result.Success {
		status = models.SpeedTaskFailed
	}
	err := models.DB.Model(&task).Updates(map[string]any{
		"status":        status,
		"latency_ms":    result.LatencyMs,
		"download_mbps": result.DownloadMbps,
		"test_bytes":    result.TestBytes,
		"egress_ip":     result.EgressIP,
		"error_message": result.Error,
		"completed_at":  &now,
	}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "00000", "msg": "结果已上报"})
}
