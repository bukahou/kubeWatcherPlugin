// atlhyper_master_v2/database/sqlite/dialect.go
// SQLite Dialect 实现
package sqlite

import (
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

// Dialect SQLite 方言
type Dialect struct {
	audit           *auditDialect
	user            *userDialect
	event           *eventDialect
	notify          *notifyDialect
	cluster         *clusterDialect
	command         *commandDialect
	settings        *settingsDialect
	aiConversation  *aiConversationDialect
	aiMessage       *aiMessageDialect
	aiProvider      *aiProviderDialect
	aiSettings      *aiSettingsDialect
	aiProviderModel *aiProviderModelDialect
	slo             *sloDialect
	aiRoleBudget    *aiRoleBudgetDialect
	aiReport        *aiReportDialect

	aiopsBaseline *aIOpsBaselineDialect
	aiopsGraph    *aIOpsGraphDialect
	aiopsIncident *aIOpsIncidentDialect

	gitHubInstall *gitHubInstallDialect
	repoConfig    *repoConfigDialect
	deployConfig  *deployConfigDialect
	deployHistory *deployHistoryDialect
}

// NewDialect 创建 SQLite 方言
func NewDialect() *Dialect {
	return &Dialect{
		audit:           &auditDialect{},
		user:            &userDialect{},
		event:           &eventDialect{},
		notify:          &notifyDialect{},
		cluster:         &clusterDialect{},
		command:         &commandDialect{},
		settings:        &settingsDialect{},
		aiConversation:  &aiConversationDialect{},
		aiMessage:       &aiMessageDialect{},
		aiProvider:      &aiProviderDialect{},
		aiSettings:      &aiSettingsDialect{},
		aiProviderModel: &aiProviderModelDialect{},
		slo:             &sloDialect{},
		aiRoleBudget:    &aiRoleBudgetDialect{},
		aiReport:        &aiReportDialect{},

		aiopsBaseline: &aIOpsBaselineDialect{},
		aiopsGraph:    &aIOpsGraphDialect{},
		aiopsIncident: &aIOpsIncidentDialect{},

		gitHubInstall: &gitHubInstallDialect{},
		repoConfig:    &repoConfigDialect{},
		deployConfig:  &deployConfigDialect{},
		deployHistory: &deployHistoryDialect{},
	}
}

func (d *Dialect) Audit() database.AuditDialect                     { return d.audit }
func (d *Dialect) User() database.UserDialect                       { return d.user }
func (d *Dialect) Event() database.EventDialect                     { return d.event }
func (d *Dialect) Notify() database.NotifyDialect                   { return d.notify }
func (d *Dialect) Cluster() database.ClusterDialect                 { return d.cluster }
func (d *Dialect) Command() database.CommandDialect                 { return d.command }
func (d *Dialect) Settings() database.SettingsDialect               { return d.settings }
func (d *Dialect) AIConversation() database.AIConversationDialect   { return d.aiConversation }
func (d *Dialect) AIMessage() database.AIMessageDialect             { return d.aiMessage }
func (d *Dialect) AIProvider() database.AIProviderDialect           { return d.aiProvider }
func (d *Dialect) AISettings() database.AISettingsDialect           { return d.aiSettings }
func (d *Dialect) AIProviderModel() database.AIProviderModelDialect { return d.aiProviderModel }
func (d *Dialect) SLO() database.SLODialect                         { return d.slo }
func (d *Dialect) AIRoleBudget() database.AIRoleBudgetDialect       { return d.aiRoleBudget }
func (d *Dialect) AIReport() database.AIReportDialect               { return d.aiReport }

func (d *Dialect) AIOpsBaseline() database.AIOpsBaselineDialect { return d.aiopsBaseline }
func (d *Dialect) AIOpsGraph() database.AIOpsGraphDialect       { return d.aiopsGraph }
func (d *Dialect) AIOpsIncident() database.AIOpsIncidentDialect { return d.aiopsIncident }

func (d *Dialect) GitHubInstall() database.GitHubInstallDialect { return d.gitHubInstall }
func (d *Dialect) RepoConfig() database.RepoConfigDialect       { return d.repoConfig }
func (d *Dialect) DeployConfig() database.DeployConfigDialect   { return d.deployConfig }
func (d *Dialect) DeployHistory() database.DeployHistoryDialect { return d.deployHistory }
func (d *Dialect) Migrate(db *sql.DB) error {
	return migrate(db)
}

// 确保实现了接口
var _ database.Dialect = (*Dialect)(nil)
