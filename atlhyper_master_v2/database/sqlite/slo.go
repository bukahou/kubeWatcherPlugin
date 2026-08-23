// atlhyper_master_v2/database/sqlite/slo.go
// SQLite SLO Dialect 实现（目标配置 + 路由映射）
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type sloDialect struct{}

// ==================== Targets ====================

func (d *sloDialect) SelectTargets(clusterID string) (string, []any) {
	return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, tls, window_days, availability_target, p95_latency_target, created_at, updated_at
		FROM slo_targets WHERE cluster_id = ? ORDER BY host`, []any{clusterID}
}

func (d *sloDialect) SelectTargetsByHost(clusterID, host string) (string, []any) {
	return `SELECT id, cluster_id, host, ingress_name, ingress_class, namespace, tls, window_days, availability_target, p95_latency_target, created_at, updated_at
		FROM slo_targets WHERE cluster_id = ? AND host = ?`, []any{clusterID, host}
}

func (d *sloDialect) UpsertTarget(t *database.SLOTarget) (string, []any) {
	query := `INSERT INTO slo_targets (cluster_id, host, ingress_name, ingress_class, namespace, tls, window_days, availability_target, p95_latency_target, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, host) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		ingress_class = excluded.ingress_class,
		namespace = excluded.namespace,
		tls = excluded.tls,
		availability_target = excluded.availability_target,
		p95_latency_target = excluded.p95_latency_target,
		updated_at = excluded.updated_at`
	args := []any{
		t.ClusterID, t.Host, t.IngressName, t.IngressClass, t.Namespace,
		boolToInt(t.TLS), t.WindowDays, t.AvailabilityTarget, t.P95LatencyTarget,
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) DeleteTarget(clusterID, host, timeRange string) (string, []any) {
	return `DELETE FROM slo_targets WHERE cluster_id = ? AND host = ?`,
		[]any{clusterID, host, timeRange}
}

func (d *sloDialect) ScanTarget(rows *sql.Rows) (*database.SLOTarget, error) {
	t := &database.SLOTarget{}
	var tls int
	var createdAt, updatedAt string
	err := rows.Scan(&t.ID, &t.ClusterID, &t.Host, &t.IngressName, &t.IngressClass,
		&t.Namespace, &tls, &t.WindowDays, &t.AvailabilityTarget, &t.P95LatencyTarget,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	t.TLS = tls == 1
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return t, nil
}

// ==================== Route Mapping ====================

func (d *sloDialect) UpsertRouteMapping(m *database.SLORouteMapping) (string, []any) {
	query := `INSERT INTO slo_route_mapping (cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, domain, path_prefix) DO UPDATE SET
		ingress_name = excluded.ingress_name,
		namespace = excluded.namespace,
		tls = excluded.tls,
		service_key = excluded.service_key,
		service_name = excluded.service_name,
		service_port = excluded.service_port,
		updated_at = excluded.updated_at`
	args := []any{
		m.ClusterID, m.Domain, m.PathPrefix, m.IngressName, m.Namespace,
		boolToInt(m.TLS), m.ServiceKey, m.ServiceName, m.ServicePort,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *sloDialect) SelectRouteMappingByServiceKey(clusterID, serviceKey string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? AND service_key = ?`,
		[]any{clusterID, serviceKey}
}

func (d *sloDialect) SelectRouteMappingsByDomain(clusterID, domain string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? AND domain = ? ORDER BY path_prefix`,
		[]any{clusterID, domain}
}

func (d *sloDialect) SelectAllRouteMappings(clusterID string) (string, []any) {
	return `SELECT id, cluster_id, domain, path_prefix, ingress_name, namespace, tls, service_key, service_name, service_port, created_at, updated_at
		FROM slo_route_mapping WHERE cluster_id = ? ORDER BY domain, path_prefix`,
		[]any{clusterID}
}

func (d *sloDialect) SelectAllDomains(clusterID string) (string, []any) {
	return `SELECT DISTINCT domain FROM slo_route_mapping WHERE cluster_id = ? ORDER BY domain`,
		[]any{clusterID}
}

func (d *sloDialect) DeleteRouteMapping(clusterID, serviceKey string) (string, []any) {
	return `DELETE FROM slo_route_mapping WHERE cluster_id = ? AND service_key = ?`,
		[]any{clusterID, serviceKey}
}

func (d *sloDialect) ScanRouteMapping(rows *sql.Rows) (*database.SLORouteMapping, error) {
	m := &database.SLORouteMapping{}
	var tls int
	var createdAt, updatedAt string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Domain, &m.PathPrefix, &m.IngressName,
		&m.Namespace, &tls, &m.ServiceKey, &m.ServiceName, &m.ServicePort,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.TLS = tls == 1
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return m, nil
}

var _ database.SLODialect = (*sloDialect)(nil)
