package cron

import (
	"context"
	"time"

	"go.uber.org/fx"
)

type Job interface {
	Name() string
	Run(ctx context.Context) error
}

type ScheduleKind string

const (
	ScheduleCron     ScheduleKind = "cron"
	ScheduleDuration ScheduleKind = "duration"
)

type ScheduleSpec struct {
	Kind ScheduleKind

	// Kind == ScheduleCron 时使用，支持 5 段或 6 段（由 WithSeconds 控制）
	Cron        string
	WithSeconds bool

	// Kind == ScheduleDuration 时使用
	Duration time.Duration

	// 单个任务运行超时；<=0 时使用默认值
	Timeout time.Duration

	// 防止同一任务重入
	Singleton bool

	// 便于后续按标签管理
	Tags []string
}

type JobRegistration struct {
	Job      Job
	Schedule ScheduleSpec
}

// AsJobRegistration annotates a constructor so its result is provided to the "jobs" value group.
// Use with fx.Provide(AsJobRegistration(domain.NewJobRegistration)).
func AsJobRegistration(f any) any {
	return fx.Annotate(
		f,
		fx.ResultTags(`group:"jobs"`),
	)
}
