package model

import "time"

type User struct {
	ID          int
	Username    string
	PasswordHash string
	DisplayName string
	Email       string
	Role        int
	CreatedAt   time.Time
	LastLogin   *time.Time
}

type GetUserAuditLogsResponse struct {
	ID        int
	UserID    int
	Username  string
	Role      int
	Action    string
	Success   bool
	Timestamp string
}