// atlhyper_master_v2/service/query/user.go
// QueryUser 实现 — 用户读取
//
// 2026-08-29：此前 admin/user.go 直接持有 database.UserRepository，
// 绕过 service 层（违反 CLAUDE.md「Gateway 直接访问 Database」禁令）。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

func (q *QueryService) GetUserByID(ctx context.Context, id int64) (*database.User, error) {
	return q.userRepo.GetByID(ctx, id)
}

func (q *QueryService) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	return q.userRepo.GetByUsername(ctx, username)
}

func (q *QueryService) ListUsers(ctx context.Context) ([]*database.User, error) {
	return q.userRepo.List(ctx)
}
