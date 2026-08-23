// atlhyper_master_v2/database/sqlite/ai_report.go
// AIReport SQLite Dialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiReportDialect struct{}

func (d *aiReportDialect) Insert(r *database.AIReport) (string, []any) {
	query := `INSERT INTO ai_reports
		(incident_id, cluster_id, role, trigger,
		 summary, root_cause_analysis, recommendations, similar_incidents,
		 investigation_steps, evidence_chain,
		 provider_name, model, input_tokens, output_tokens, duration_ms,
		 created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		nullString(r.IncidentID), r.ClusterID, r.Role, r.Trigger,
		r.Summary, r.RootCauseAnalysis, r.Recommendations, r.SimilarIncidents,
		r.InvestigationSteps, r.EvidenceChain,
		r.ProviderName, r.Model, r.InputTokens, r.OutputTokens, r.DurationMs,
		r.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *aiReportDialect) SelectByID(id int64) (string, []any) {
	return selectAIReportColumns + ` FROM ai_reports WHERE id = ?`, []any{id}
}

func (d *aiReportDialect) SelectByIncident(incidentID string) (string, []any) {
	return selectAIReportColumns + ` FROM ai_reports WHERE incident_id = ? ORDER BY created_at DESC`, []any{incidentID}
}

func (d *aiReportDialect) SelectByCluster(clusterID, role string, limit int) (string, []any) {
	if role != "" {
		return selectAIReportColumns + ` FROM ai_reports WHERE cluster_id = ? AND role = ? ORDER BY created_at DESC LIMIT ?`,
			[]any{clusterID, role, limit}
	}
	return selectAIReportColumns + ` FROM ai_reports WHERE cluster_id = ? ORDER BY created_at DESC LIMIT ?`,
		[]any{clusterID, limit}
}

func (d *aiReportDialect) SelectRecent(role string, limit, offset int) (string, []any) {
	if role != "" {
		return selectAIReportColumns + ` FROM ai_reports WHERE role = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			[]any{role, limit, offset}
	}
	return selectAIReportColumns + ` FROM ai_reports ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		[]any{limit, offset}
}

func (d *aiReportDialect) CountRecent(role string) (string, []any) {
	if role != "" {
		return `SELECT COUNT(*) FROM ai_reports WHERE role = ?`, []any{role}
	}
	return `SELECT COUNT(*) FROM ai_reports`, nil
}

func (d *aiReportDialect) UpdateResult(id int64, r *database.AIReport) (string, []any) {
	return `UPDATE ai_reports SET
		summary = ?, root_cause_analysis = ?, recommendations = ?,
		investigation_steps = ?,
		input_tokens = ?, output_tokens = ?, duration_ms = ?
	WHERE id = ?`, []any{
			r.Summary, r.RootCauseAnalysis, r.Recommendations,
			r.InvestigationSteps,
			r.InputTokens, r.OutputTokens, r.DurationMs,
			id,
		}
}

func (d *aiReportDialect) CountByClusterAndRole(clusterID, role string, since time.Time) (string, []any) {
	return `SELECT COUNT(*) FROM ai_reports WHERE cluster_id = ? AND role = ? AND created_at >= ?`,
		[]any{clusterID, role, since.Format(time.RFC3339)}
}

func (d *aiReportDialect) DeleteBefore(before time.Time) (string, []any) {
	return `DELETE FROM ai_reports WHERE created_at < ?`, []any{before.Format(time.RFC3339)}
}

func (d *aiReportDialect) UpdateInvestigationSteps(id int64, steps string) (string, []any) {
	return `UPDATE ai_reports SET investigation_steps = ? WHERE id = ?`, []any{steps, id}
}

func (d *aiReportDialect) ScanRow(rows *sql.Rows) (*database.AIReport, error) {
	r := &database.AIReport{}
	var incidentID sql.NullString
	var createdAt string
	var summary, rootCause, recommendations, similar sql.NullString
	var steps, evidence sql.NullString
	var providerName, model sql.NullString

	err := rows.Scan(&r.ID, &incidentID, &r.ClusterID, &r.Role, &r.Trigger,
		&summary, &rootCause, &recommendations, &similar,
		&steps, &evidence,
		&providerName, &model, &r.InputTokens, &r.OutputTokens, &r.DurationMs,
		&createdAt)
	if err != nil {
		return nil, err
	}

	if incidentID.Valid {
		r.IncidentID = incidentID.String
	}
	if summary.Valid {
		r.Summary = summary.String
	}
	if rootCause.Valid {
		r.RootCauseAnalysis = rootCause.String
	}
	if recommendations.Valid {
		r.Recommendations = recommendations.String
	}
	if similar.Valid {
		r.SimilarIncidents = similar.String
	}
	if steps.Valid {
		r.InvestigationSteps = steps.String
	}
	if evidence.Valid {
		r.EvidenceChain = evidence.String
	}
	if providerName.Valid {
		r.ProviderName = providerName.String
	}
	if model.Valid {
		r.Model = model.String
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return r, nil
}

// selectAIReportColumns 共用的 SELECT 列列表
const selectAIReportColumns = `SELECT id, incident_id, cluster_id, role, trigger,
	summary, root_cause_analysis, recommendations, similar_incidents,
	investigation_steps, evidence_chain,
	provider_name, model, input_tokens, output_tokens, duration_ms,
	created_at`

// nullString 将空字符串转为 sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

var _ database.AIReportDialect = (*aiReportDialect)(nil)
