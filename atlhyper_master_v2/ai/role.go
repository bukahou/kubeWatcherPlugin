// atlhyper_master_v2/ai/role.go
// AI 角色常量、路由配置与解析
package ai

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// 角色常量
const (
	RoleBackground = "background" // 后台分析（AIOps 事件摘要）
	RoleChat       = "chat"       // 交互对话（用户 Chat）
	RoleAnalysis   = "analysis"   // 深度分析（高危事件调查）
)

// ValidRoles 有效角色列表
var ValidRoles = []string{RoleBackground, RoleChat, RoleAnalysis}

// IsValidRole 检查角色是否有效
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// RoleConfig 角色路由解析结果
type RoleConfig struct {
	llm.Config
	ContextWindow int    // 有效上下文窗口（模型默认值 or 用户覆盖值）
	ProviderID    int64  // Provider ID（用于统计）
	ProviderName  string // Provider 名称（用于日志/报告）
}

// loadAIConfigForRole 按角色加载 AI 配置
// 解析优先级:
//  1. 查找持有该角色的 Provider → 检查预算 → 返回
//  2. 预算耗尽 → 使用 fallback Provider
//  3. 无角色分配 → 返回错误
func (s *aiServiceImpl) loadAIConfigForRole(ctx context.Context, role string) (*RoleConfig, error) {
	// 1. 查找持有该角色的 Provider
	providers, _ := s.providerRepo.List(ctx)
	for _, p := range providers {
		if !containsRole(p.Roles, role) {
			continue
		}

		// 找到了持有该角色的 Provider → 检查预算（含跨日/跨月重置）
		if s.budgetRepo != nil {
			if budget, _ := s.budgetRepo.Get(ctx, role); budget != nil {
				// 跨日重置
				if needsDailyReset(budget) {
					if err := s.budgetRepo.ResetDailyUsage(ctx, role); err != nil {
						log.Warn("跨日重置失败", "role", role, "err", err)
					} else {
						budget.DailyInputTokensUsed = 0
						budget.DailyOutputTokensUsed = 0
						budget.DailyCallsUsed = 0
					}
				}
				// 跨月重置
				if needsMonthlyReset(budget) {
					if err := s.budgetRepo.ResetMonthlyUsage(ctx, role); err != nil {
						log.Warn("跨月重置失败", "role", role, "err", err)
					} else {
						budget.MonthlyInputTokensUsed = 0
						budget.MonthlyOutputTokensUsed = 0
						budget.MonthlyCallsUsed = 0
					}
				}

				if !checkBudget(budget) {
					// 预算耗尽 → 尝试 fallback
					if budget.FallbackProviderID != nil {
						fallback, err := s.providerRepo.GetByID(ctx, *budget.FallbackProviderID)
						if err == nil && fallback != nil {
							log.Warn("角色预算耗尽，使用降级 Provider",
								"role", role, "fallback", fallback.Name)
							return s.providerToRoleConfig(ctx, fallback), nil
						}
					}
					return nil, fmt.Errorf("角色 %s 预算已用尽", role)
				}
			}
		}

		return s.providerToRoleConfig(ctx, p), nil
	}

	// 3. 无角色分配 → 严格模式：返回错误
	return nil, fmt.Errorf("角色 %s 未分配 Provider，请在 AI 设置中为该角色指定提供商", role)
}

// providerToRoleConfig 将 Provider 转换为 RoleConfig
func (s *aiServiceImpl) providerToRoleConfig(ctx context.Context, p *database.AIProvider) *RoleConfig {
	cfg := &RoleConfig{
		Config: llm.Config{
			Provider: p.Provider,
			APIKey:   p.APIKey,
			Model:    p.Model,
			BaseURL:  p.BaseURL,
		},
		ProviderID:   p.ID,
		ProviderName: p.Name,
	}

	// 解析有效上下文窗口
	cfg.ContextWindow = EffectiveContextWindow(ctx, p, s.modelRepo)
	return cfg
}

// EffectiveContextWindow 获取 Provider 的有效 context_window
func EffectiveContextWindow(ctx context.Context, provider *database.AIProvider, modelRepo database.AIProviderModelRepository) int {
	// 用户覆盖优先
	if provider.ContextWindowOverride > 0 {
		return provider.ContextWindowOverride
	}
	// 查模型默认值
	if modelRepo != nil {
		models, _ := modelRepo.ListByProvider(ctx, provider.Provider)
		for _, m := range models {
			if m.Model == provider.Model && m.ContextWindow > 0 {
				return m.ContextWindow
			}
		}
	}
	return 0
}

// checkBudget 检查预算是否还有余额（多维度：input/output × 日/月）
func checkBudget(budget *database.AIRoleBudget) bool {
	// 日限额检查
	if budget.DailyInputTokenLimit > 0 && budget.DailyInputTokensUsed >= budget.DailyInputTokenLimit {
		return false
	}
	if budget.DailyOutputTokenLimit > 0 && budget.DailyOutputTokensUsed >= budget.DailyOutputTokenLimit {
		return false
	}
	if budget.DailyCallLimit > 0 && budget.DailyCallsUsed >= budget.DailyCallLimit {
		return false
	}

	// 月限额检查
	if budget.MonthlyInputTokenLimit > 0 && budget.MonthlyInputTokensUsed >= budget.MonthlyInputTokenLimit {
		return false
	}
	if budget.MonthlyOutputTokenLimit > 0 && budget.MonthlyOutputTokensUsed >= budget.MonthlyOutputTokenLimit {
		return false
	}
	if budget.MonthlyCallLimit > 0 && budget.MonthlyCallsUsed >= budget.MonthlyCallLimit {
		return false
	}

	return true
}

// needsDailyReset 检查是否需要跨日重置
func needsDailyReset(budget *database.AIRoleBudget) bool {
	if budget.DailyResetAt == nil {
		return false
	}
	resetDate := budget.DailyResetAt.Truncate(24 * time.Hour)
	today := time.Now().Truncate(24 * time.Hour)
	return today.After(resetDate)
}

// needsMonthlyReset 检查是否需要跨月重置
func needsMonthlyReset(budget *database.AIRoleBudget) bool {
	if budget.MonthlyResetAt == nil {
		return false
	}
	resetYear, resetMonth, _ := budget.MonthlyResetAt.Date()
	nowYear, nowMonth, _ := time.Now().Date()
	return nowYear > resetYear || nowMonth > resetMonth
}

// RecordUsage 记录 AI 调用消耗（预算扣减 + Provider 统计更新）
func (s *aiServiceImpl) RecordUsage(ctx context.Context, role string, providerID int64, inputTokens, outputTokens int) {
	// 1. 扣减角色预算（日 + 月同时扣减）
	if s.budgetRepo != nil {
		if err := s.budgetRepo.IncrementUsage(ctx, role, inputTokens, outputTokens); err != nil {
			log.Warn("扣减角色预算失败", "role", role, "err", err)
		}
	}
	// 2. 累加 Provider 统计
	totalTokens := int64(inputTokens + outputTokens)
	if err := s.providerRepo.IncrementUsage(ctx, providerID, 1, totalTokens, 0); err != nil {
		log.Warn("更新 Provider 统计失败", "provider", providerID, "err", err)
	}
}

// containsRole 检查角色列表中是否包含指定角色
func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// MaxPromptCharsForContext 根据 context_window 计算 Prompt 最大字符数
// 供 AIOps Enhancer 使用
func MaxPromptCharsForContext(contextWindow int) int {
	if contextWindow <= 0 {
		return 16000 // 云端大模型: 保持默认值
	}
	// 上下文窗口的 50% 给 prompt（另 50% 给输出 + overhead）
	// 1 token ≈ 2.5 chars
	chars := contextWindow / 2 * 25 / 10
	if chars < 2000 {
		return 2000
	}
	return chars
}
