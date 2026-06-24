package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const (
	SpeedTaskPending = "pending"
	SpeedTaskRunning = "running"
	SpeedTaskSuccess = "success"
	SpeedTaskFailed  = "failed"
)

type HomeAgent struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	Name             string     `json:"name"`
	TokenHash        string     `json:"-"`
	PersistentActive bool       `json:"persistent_active"`
	LastSeen         *time.Time `json:"last_seen"`
	AgentVersion     string     `json:"agent_version"`
	Platform         string     `json:"platform"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type SpeedTestTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	HomeAgentID  uint       `json:"home_agent_id"`
	NodeID       int        `json:"node_id"`
	NodeName     string     `json:"node_name"`
	TestType     string     `json:"test_type"`
	Status       string     `json:"status"`
	NodeLink     string     `json:"-"`
	LatencyMs    int64      `json:"latency_ms"`
	DownloadMbps float64    `json:"download_mbps"`
	TestBytes    int64      `json:"test_bytes"`
	EgressIP     string     `json:"egress_ip"`
	ErrorMessage string     `json:"error_message"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func HashAgentToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
