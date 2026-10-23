package entities

import (
	"time"
)

type AgentExecutionMode string

const (
	AgentModeManaged     AgentExecutionMode = "MANAGED"
	AgentModeByoaWebhook AgentExecutionMode = "BYOA_WEBHOOK"
	AgentModeByoLLM      AgentExecutionMode = "BYO_LLM"
)

type RemediationAutonomyLevel string

const (
	RemediationAdvisory         RemediationAutonomyLevel = "ADVISORY"
	RemediationApprovalRequired RemediationAutonomyLevel = "APPROVAL_REQUIRED"
	RemediationAutonomous       RemediationAutonomyLevel = "AUTONOMOUS"
)

type ExternalMCPServerConfig struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Transport  string `json:"transport"` // "http-jsonrpc" or "sse"
	AuthHeader string `json:"authHeader,omitempty"`
	Enabled    bool   `json:"enabled"`
}

type RemediationPlaybookConfig struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	TriggerPattern   string            `json:"triggerPattern"`
	ActionType       string            `json:"actionType"` // "WEBHOOK_CALL", "GITHUB_WORKFLOW", "K8S_ROLLOUT_RESTART"
	TargetURL        string            `json:"targetUrl"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers,omitempty"`
	PayloadTemplate  string            `json:"payloadTemplate,omitempty"`
	RequiresApproval bool              `json:"requiresApproval"`
}

type AgentConfig struct {
	ID                     uint                        `gorm:"primaryKey" json:"id"`
	TeamID                 uint                        `gorm:"uniqueIndex;not null" json:"teamId"`
	Enabled                bool                        `gorm:"not null;default:true" json:"enabled"`
	ExecutionMode          AgentExecutionMode          `gorm:"not null;default:'MANAGED'" json:"executionMode"`
	AutoTriggerOnConsensus bool                        `gorm:"not null;default:true" json:"autoTriggerOnConsensus"`
	AutoTriggerOnAnomaly   bool                        `gorm:"not null;default:true" json:"autoTriggerOnAnomaly"`
	AutoTriggerOnLogSpike  bool                        `gorm:"not null;default:true" json:"autoTriggerOnLogSpike"`
	MinOccurrences         int                         `gorm:"not null;default:1" json:"minOccurrences"`
	CooldownMinutes        int                         `gorm:"not null;default:30" json:"cooldownMinutes"`
	RemediationLevel       RemediationAutonomyLevel    `gorm:"not null;default:'APPROVAL_REQUIRED'" json:"remediationLevel"`
	ByoaWebhookURL         *string                     `json:"byoaWebhookUrl,omitempty"`
	ByoaWebhookSecret      *string                     `json:"byoaWebhookSecret,omitempty"`
	CustomLLMProvider      *string                     `json:"customLlmProvider,omitempty"`
	CustomLLMBaseURL       *string                     `json:"customLlmBaseUrl,omitempty"`
	CustomLLMModel         *string                     `json:"customLlmModel,omitempty"`
	CustomLLMAPIKey        *string                     `json:"customLlmApiKey,omitempty"`
	ExternalMCPServers     []ExternalMCPServerConfig   `gorm:"serializer:json" json:"externalMcpServers"`
	RemediationPlaybooks   []RemediationPlaybookConfig `gorm:"serializer:json" json:"remediationPlaybooks"`
	CreatedAt              time.Time                   `gorm:"index" json:"createdAt"`
	UpdatedAt              time.Time                   `gorm:"index" json:"updatedAt"`
}

func (AgentConfig) TableName() string {
	return "agent_configs"
}
