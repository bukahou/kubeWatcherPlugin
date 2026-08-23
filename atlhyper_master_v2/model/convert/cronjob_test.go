package convert

import (
	"testing"
	"time"

	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

func TestCronJobItem_FieldMapping(t *testing.T) {
	lastSchedule := time.Date(2026, 2, 14, 2, 0, 0, 0, time.UTC)
	lastSuccess := time.Date(2026, 2, 14, 2, 5, 0, 0, time.UTC)
	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			Name:      "backup-daily",
			Namespace: "default",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Schedule:           "0 2 * * *",
		Suspend:            false,
		ActiveJobs:         0,
		LastScheduleTime:   &lastSchedule,
		LastSuccessfulTime: &lastSuccess,
	}

	item := CronJobItem(src)

	if item.Name != "backup-daily" {
		t.Errorf("Name = %q, want %q", item.Name, "backup-daily")
	}
	if item.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "default")
	}
	if item.Schedule != "0 2 * * *" {
		t.Errorf("Schedule = %q, want %q", item.Schedule, "0 2 * * *")
	}
	if item.Suspend {
		t.Error("Suspend = true, want false")
	}
	if item.ActiveJobs != 0 {
		t.Errorf("ActiveJobs = %d, want 0", item.ActiveJobs)
	}
	if item.LastScheduleTime != "2026-02-14T02:00:00Z" {
		t.Errorf("LastScheduleTime = %q, want %q", item.LastScheduleTime, "2026-02-14T02:00:00Z")
	}
	if item.LastSuccessfulTime != "2026-02-14T02:05:00Z" {
		t.Errorf("LastSuccessfulTime = %q, want %q", item.LastSuccessfulTime, "2026-02-14T02:05:00Z")
	}
	if item.Age == "" {
		t.Error("Age should not be empty")
	}
}

func TestCronJobItem_NilTimePointers(t *testing.T) {
	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			Name:      "new-cronjob",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Schedule:           "*/5 * * * *",
		LastScheduleTime:   nil,
		LastSuccessfulTime: nil,
	}

	item := CronJobItem(src)

	if item.LastScheduleTime != "" {
		t.Errorf("LastScheduleTime = %q, want empty", item.LastScheduleTime)
	}
	if item.LastSuccessfulTime != "" {
		t.Errorf("LastSuccessfulTime = %q, want empty", item.LastSuccessfulTime)
	}
}

func TestCronJobItems_NilInput(t *testing.T) {
	result := CronJobItems(nil)
	if result == nil {
		t.Error("CronJobItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestCronJobItems_EmptyInput(t *testing.T) {
	result := CronJobItems([]cluster.CronJob{})
	if result == nil {
		t.Error("CronJobItems([]) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

// ============================================================
// CronJobDetail 测试
// ============================================================

func TestCronJobDetail_FieldMapping(t *testing.T) {
	lastSchedule := time.Date(2026, 2, 14, 2, 0, 0, 0, time.UTC)
	lastSuccess := time.Date(2026, 2, 14, 2, 5, 0, 0, time.UTC)
	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			UID:       "cron-456",
			Name:      "backup-daily",
			Namespace: "default",
			OwnerKind: "",
			OwnerName: "",
			Labels:    map[string]string{"tier": "infra"},
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Schedule:           "0 2 * * *",
		Suspend:            false,
		ActiveJobs:         1,
		LastScheduleTime:   &lastSchedule,
		LastSuccessfulTime: &lastSuccess,
	}

	detail := CronJobDetail(src)

	if detail.UID != "cron-456" {
		t.Errorf("UID = %q, want %q", detail.UID, "cron-456")
	}
	if detail.Name != "backup-daily" {
		t.Errorf("Name = %q, want %q", detail.Name, "backup-daily")
	}
	if detail.Schedule != "0 2 * * *" {
		t.Errorf("Schedule = %q, want %q", detail.Schedule, "0 2 * * *")
	}
	if detail.Suspend {
		t.Error("Suspend = true, want false")
	}
	if detail.ActiveJobs != 1 {
		t.Errorf("ActiveJobs = %d, want 1", detail.ActiveJobs)
	}
	if detail.LastScheduleTime != "2026-02-14T02:00:00Z" {
		t.Errorf("LastScheduleTime = %q, want %q", detail.LastScheduleTime, "2026-02-14T02:00:00Z")
	}
	if detail.LastSuccessfulTime != "2026-02-14T02:05:00Z" {
		t.Errorf("LastSuccessfulTime = %q, want %q", detail.LastSuccessfulTime, "2026-02-14T02:05:00Z")
	}
	if detail.Labels["tier"] != "infra" {
		t.Errorf("Labels[tier] = %q, want %q", detail.Labels["tier"], "infra")
	}
}

func TestCronJobDetail_AgoFields(t *testing.T) {
	// 10 分钟前
	tenMinsAgo := time.Now().Add(-10 * time.Minute)
	// 3 小时前
	threeHoursAgo := time.Now().Add(-3 * time.Hour)

	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			Name:      "test-cron",
			Namespace: "default",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		Schedule:           "*/5 * * * *",
		LastScheduleTime:   &tenMinsAgo,
		LastSuccessfulTime: &threeHoursAgo,
	}

	detail := CronJobDetail(src)

	if detail.LastScheduleAgo != "10m" {
		t.Errorf("LastScheduleAgo = %q, want %q", detail.LastScheduleAgo, "10m")
	}
	if detail.LastSuccessAgo != "3h" {
		t.Errorf("LastSuccessAgo = %q, want %q", detail.LastSuccessAgo, "3h")
	}
}

func TestCronJobDetail_NilTimes(t *testing.T) {
	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			Name:      "new-cron",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Schedule:           "*/5 * * * *",
		LastScheduleTime:   nil,
		LastSuccessfulTime: nil,
	}

	detail := CronJobDetail(src)

	if detail.LastScheduleTime != "" {
		t.Errorf("LastScheduleTime = %q, want empty", detail.LastScheduleTime)
	}
	if detail.LastSuccessfulTime != "" {
		t.Errorf("LastSuccessfulTime = %q, want empty", detail.LastSuccessfulTime)
	}
	if detail.LastScheduleAgo != "" {
		t.Errorf("LastScheduleAgo = %q, want empty", detail.LastScheduleAgo)
	}
	if detail.LastSuccessAgo != "" {
		t.Errorf("LastSuccessAgo = %q, want empty", detail.LastSuccessAgo)
	}
}

func TestCronJobDetail_SpecFields(t *testing.T) {
	successLimit := int32(3)
	failedLimit := int32(1)
	src := &cluster.CronJob{
		CommonMeta: model_v3.CommonMeta{
			Name:      "test-cron",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Schedule:                   "*/5 * * * *",
		ConcurrencyPolicy:          "Forbid",
		SuccessfulJobsHistoryLimit: &successLimit,
		FailedJobsHistoryLimit:     &failedLimit,
		Template: cluster.PodTemplate{
			Containers: []cluster.ContainerDetail{
				{Name: "worker", Image: "busybox:latest"},
			},
		},
	}

	detail := CronJobDetail(src)

	if detail.ConcurrencyPolicy != "Forbid" {
		t.Errorf("ConcurrencyPolicy = %q, want %q", detail.ConcurrencyPolicy, "Forbid")
	}
	if detail.SuccessfulJobsHistoryLimit == nil || *detail.SuccessfulJobsHistoryLimit != 3 {
		t.Errorf("SuccessfulJobsHistoryLimit = %v, want 3", detail.SuccessfulJobsHistoryLimit)
	}
	if detail.FailedJobsHistoryLimit == nil || *detail.FailedJobsHistoryLimit != 1 {
		t.Errorf("FailedJobsHistoryLimit = %v, want 1", detail.FailedJobsHistoryLimit)
	}
	if detail.Template == nil {
		t.Error("Template should not be nil when containers exist")
	}
}
