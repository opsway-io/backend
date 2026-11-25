package investigation

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

// reserveManagedQuotaOrFallback enforces concurrency, monthly token budget, and atomic run quota
// for MANAGED investigations (never bypassable via Force). Falls back to BYO-LLM / BYOA if configured.
func (s *serviceImpl) reserveManagedQuotaOrFallback(
	ctx context.Context,
	teamID uint,
	agentCfg *entities.AgentConfig,
) (entities.AgentExecutionMode, error) {
	effectiveMode := agentCfg.ExecutionMode

	var team entities.Team
	if err := s.db.WithContext(ctx).Select("id, payment_plan, allow_overage").First(&team, teamID).Error; err != nil {
		team.PaymentPlan = entities.PaymentPlanFree
	}

	// 1. Concurrency cap (prevents burst / race-condition compute abuse across all execution modes)
	var inFlight int64
	s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
		Where("team_id = ? AND status IN ?", teamID, []entities.InvestigationStatus{
			entities.InvestigationStatusQueued,
			entities.InvestigationStatusRunning,
		}).
		Count(&inFlight)
	if int(inFlight) >= team.GetMaxConcurrentInvestigations() {
		return effectiveMode, ErrQuotaExceeded
	}

	if effectiveMode != entities.AgentModeManaged {
		return effectiveMode, nil
	}

	allowPaidOverage := team.AllowOverage && (team.PaymentPlan == entities.PaymentPlanTeam || team.PaymentPlan == entities.PaymentPlanEnterprise)

	tryBYOFallback := func() (entities.AgentExecutionMode, error) {
		if agentCfg.CustomLLMAPIKey != nil && *agentCfg.CustomLLMAPIKey != "" && agentCfg.CustomLLMBaseURL != nil && *agentCfg.CustomLLMBaseURL != "" {
			return entities.AgentModeByoLLM, nil
		}
		if agentCfg.ByoaWebhookURL != nil && *agentCfg.ByoaWebhookURL != "" {
			return entities.AgentModeByoaWebhook, nil
		}
		return effectiveMode, ErrQuotaExceeded
	}

	// 2. Monthly LLM token budget check
	monthlyTokenLimit := team.GetAgentMonthlyTokenLimit()
	if monthlyTokenLimit > 0 && !allowPaidOverage {
		now := time.Now().UTC()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		var monthTokens int64
		s.db.WithContext(ctx).Model(&entities.IncidentInvestigation{}).
			Select("COALESCE(SUM(tokens_used), 0)").
			Where("team_id = ? AND execution_mode = ? AND created_at >= ?", teamID, entities.AgentModeManaged, monthStart).
			Scan(&monthTokens)
		if monthTokens >= int64(monthlyTokenLimit) {
			return tryBYOFallback()
		}
	}

	// 3. Atomic monthly run quota check & increment (prevents concurrent race past limit)
	entitlements, _ := s.GetEntitlements(ctx, teamID)
	for _, ent := range entitlements {
		if ent.Product != entities.ProductManagedAgent {
			continue
		}
		q := s.db.WithContext(ctx).Model(&entities.TeamEntitlement{}).Where("id = ?", ent.ID)
		if ent.MonthlyQuota != -1 && !allowPaidOverage {
			q = q.Where("current_usage < monthly_quota")
		}
		res := q.UpdateColumn("current_usage", gorm.Expr("current_usage + 1"))
		if res.Error != nil {
			return effectiveMode, res.Error
		}
		if res.RowsAffected == 0 {
			return tryBYOFallback()
		}
		break
	}

	return effectiveMode, nil
}
