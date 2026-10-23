package entities

import (
	"time"
)

type InvestigationStatus string

const (
	InvestigationStatusQueued           InvestigationStatus = "QUEUED"
	InvestigationStatusRunning          InvestigationStatus = "RUNNING"
	InvestigationStatusAwaitingApproval InvestigationStatus = "AWAITING_APPROVAL"
	InvestigationStatusCompleted        InvestigationStatus = "COMPLETED"
	InvestigationStatusFailed           InvestigationStatus = "FAILED"
)

type CorrelatedLogEntry struct {
	LogID       string            `json:"logId"`
	Timestamp   string            `json:"timestamp"`
	ServiceName string            `json:"serviceName"`
	LogLevel    string            `json:"logLevel"`
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels,omitempty"`
	DeepLinkURL string            `json:"deepLinkUrl,omitempty"`
}

type InvestigationStep struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	InvestigationID uint      `gorm:"index;not null" json:"investigationId"`
	StepIndex       int       `gorm:"not null" json:"stepIndex"`
	StepType        string    `gorm:"not null" json:"stepType"` // "THOUGHT", "MCP_TOOL_CALL", "SYNTHESIS", "REMEDIATION", "VERIFICATION"
	ServerName      string    `json:"serverName,omitempty"`     // "log-search-mcp", "monitoring-mcp", "pantomath-mcp", "byoa"
	ToolName        string    `json:"toolName,omitempty"`
	InputJSON       string    `gorm:"type:text" json:"inputJson,omitempty"`
	OutputSummary   string    `gorm:"type:text" json:"outputSummary"`
	Status          string    `gorm:"not null;default:'COMPLETED'" json:"status"`
	DurationMs      int64     `json:"durationMs"`
	CreatedAt       time.Time `gorm:"index" json:"createdAt"`
}

func (InvestigationStep) TableName() string {
	return "investigation_steps"
}

type RemediationAction struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	InvestigationID     uint       `gorm:"index;not null" json:"investigationId"`
	TeamID              uint       `gorm:"index;not null" json:"teamId"`
	IncidentID          uint       `gorm:"index;not null" json:"incidentId"`
	Title               string     `gorm:"not null" json:"title"`
	Description         string     `gorm:"type:text" json:"description"`
	ActionType          string     `gorm:"not null" json:"actionType"` // "GITHUB_PR", "WEBHOOK_CALL", "K8S_ROLLOUT_RESTART", "CLI_COMMAND"
	TargetService       string     `json:"targetService,omitempty"`
	PayloadJSON         string     `gorm:"type:text" json:"payloadJson,omitempty"`
	DiffOrCommand       string     `gorm:"type:text" json:"diffOrCommand,omitempty"`
	Status              string     `gorm:"not null;default:'PENDING_APPROVAL'" json:"status"` // "SUGGESTED", "PENDING_APPROVAL", "APPROVED", "EXECUTED", "VERIFIED", "REJECTED", "FAILED"
	ApprovedByUserID    *uint      `json:"approvedByUserId,omitempty"`
	ExecutedAt          *time.Time `json:"executedAt,omitempty"`
	VerificationSummary *string    `gorm:"type:text" json:"verificationSummary,omitempty"`
	CreatedAt           time.Time  `gorm:"index" json:"createdAt"`
	UpdatedAt           time.Time  `gorm:"index" json:"updatedAt"`
}

func (RemediationAction) TableName() string {
	return "remediation_actions"
}

type IncidentInvestigation struct {
	ID                 uint                 `gorm:"primaryKey" json:"id"`
	UUID               string               `gorm:"uniqueIndex;not null" json:"uuid"`
	TeamID             uint                 `gorm:"index;not null" json:"teamId"`
	IncidentID         uint                 `gorm:"index;not null" json:"incidentId"`
	MonitorID          *uint                `gorm:"index" json:"monitorId,omitempty"`
	ServiceName        string               `json:"serviceName,omitempty"`
	TriggerSource      string               `gorm:"not null;default:'AUTOMATIC'" json:"triggerSource"` // "AUTOMATIC", "MANUAL_UI", "MANUAL_SLACK", "LOG_SPIKE", "API"
	ExecutionMode      AgentExecutionMode   `gorm:"not null;default:'MANAGED'" json:"executionMode"`
	Status             InvestigationStatus  `gorm:"index;not null;default:'RUNNING'" json:"status"`
	UserPrompt         *string              `gorm:"type:text" json:"userPrompt,omitempty"`
	RootCauseSummary   *string              `gorm:"type:text" json:"rootCauseSummary,omitempty"`
	DetailedDiagnosis  *string              `gorm:"type:text" json:"detailedDiagnosis,omitempty"`
	ConfidenceScore    *float64             `json:"confidenceScore,omitempty"`
	CorrelatedLogs     []CorrelatedLogEntry `gorm:"serializer:json" json:"correlatedLogs"`
	Steps              []InvestigationStep  `gorm:"foreignKey:InvestigationID;constraint:OnDelete:CASCADE" json:"steps"`
	RemediationActions []RemediationAction  `gorm:"foreignKey:InvestigationID;constraint:OnDelete:CASCADE" json:"remediationActions"`
	TokensUsed         int                  `gorm:"not null;default:0" json:"tokensUsed"`
	DurationMs         int64                `gorm:"not null;default:0" json:"durationMs"`
	StartedAt          time.Time            `gorm:"index" json:"startedAt"`
	CompletedAt        *time.Time           `json:"completedAt,omitempty"`
	CreatedAt          time.Time            `gorm:"index" json:"createdAt"`
	UpdatedAt          time.Time            `gorm:"index" json:"updatedAt"`
}

func (IncidentInvestigation) TableName() string {
	return "incident_investigations"
}
