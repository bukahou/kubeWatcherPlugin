// atlhyper_master_v2/service/query/deploy.go
// QueryDeploy 实现 — 部署配置、部署历史、GitHub 集成的读取
//
// 2026-08-29：此前 admin/deploy.go 与 settings/github.go 直接持有
// database repository，绕过 service 层（违反 CLAUDE.md「Gateway 直接访问
// Database」禁令）。本文件把这些读取收归 service 层。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== Deploy Config ====================

func (q *QueryService) GetDeployConfig(ctx context.Context, clusterID string) (*database.DeployConfig, error) {
	return q.deployConfigRepo.GetByCluster(ctx, clusterID)
}

// ==================== Deploy History ====================

func (q *QueryService) ListDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error) {
	return q.deployHistoryRepo.List(ctx, opts)
}

func (q *QueryService) CountDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) (int, error) {
	return q.deployHistoryRepo.Count(ctx, opts)
}

// ==================== GitHub 集成 ====================

func (q *QueryService) GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error) {
	return q.githubInstallRepo.Get(ctx)
}

func (q *QueryService) ListRepoConfigs(ctx context.Context) ([]*database.RepoConfig, error) {
	return q.repoConfigRepo.List(ctx)
}
