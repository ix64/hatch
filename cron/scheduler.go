package cron

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultJobTimeout = 30 * time.Second

type SchedulerParams struct {
	fx.In

	Logger *zap.Logger
	Jobs   []JobRegistration `group:"jobs"`
}

type zapGocronLogger struct {
	zap *zap.SugaredLogger
}

func newZapGocronLogger(logger *zap.Logger) *zapGocronLogger {
	return &zapGocronLogger{
		zap: logger.WithOptions(zap.AddCallerSkip(1)).Sugar(),
	}
}

func (l *zapGocronLogger) Debug(msg string, args ...any) { l.zap.Debugw(msg, args...) }
func (l *zapGocronLogger) Info(msg string, args ...any)  { l.zap.Infow(msg, args...) }
func (l *zapGocronLogger) Warn(msg string, args ...any)  { l.zap.Warnw(msg, args...) }
func (l *zapGocronLogger) Error(msg string, args ...any) { l.zap.Errorw(msg, args...) }

func NewScheduler(params SchedulerParams) (gocron.Scheduler, error) {
	logger := params.Logger.Named("scheduler")
	s, err := gocron.NewScheduler(gocron.WithLogger(newZapGocronLogger(logger)))
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}
	cleanupErr := func(err error) error {
		return errors.Join(err, s.Shutdown())
	}

	seen := make(map[string]struct{}, len(params.Jobs))

	for _, reg := range params.Jobs {
		if err := validateRegistration(reg); err != nil {
			return nil, cleanupErr(err)
		}

		name := reg.Job.Name()
		if _, ok := seen[name]; ok {
			return nil, cleanupErr(fmt.Errorf("duplicate job name: %s", name))
		}
		seen[name] = struct{}{}

		jobDef, task, jobOpts, err := buildGocronJob(reg, params.Logger)
		if err != nil {
			return nil, cleanupErr(fmt.Errorf("build job %s: %w", name, err))
		}

		if _, err := s.NewJob(jobDef, task, jobOpts...); err != nil {
			return nil, cleanupErr(fmt.Errorf("register job %s: %w", name, err))
		}

		logger.Info("job registered", zap.String("job", name))
	}

	return s, nil
}

func RunScheduler(lc fx.Lifecycle, scheduler gocron.Scheduler, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			scheduler.Start()
			logger.Info("scheduler started")
			return nil
		},
		OnStop: func(context.Context) error {
			if err := scheduler.Shutdown(); err != nil {
				logger.Error("scheduler shutdown failed", zap.Error(err))
				return err
			}
			logger.Info("scheduler stopped")
			return nil
		},
	})
}

func validateRegistration(reg JobRegistration) error {
	if reg.Job == nil {
		return errors.New("job is nil")
	}
	name := reg.Job.Name()
	if name == "" {
		return errors.New("job name is empty")
	}
	switch reg.Schedule.Kind {
	case ScheduleCron:
		if reg.Schedule.Cron == "" {
			return fmt.Errorf("job %s cron is empty", name)
		}
	case ScheduleDuration:
		if reg.Schedule.Duration <= 0 {
			return fmt.Errorf("job %s duration must be > 0", name)
		}
	default:
		return fmt.Errorf("job %s unknown schedule kind: %s", name, reg.Schedule.Kind)
	}
	return nil
}

func buildGocronJob(reg JobRegistration, baseLogger *zap.Logger) (gocron.JobDefinition, gocron.Task, []gocron.JobOption, error) {
	name := reg.Job.Name()
	logger := baseLogger.Named("job").With(zap.String("job", name))

	jobDef, err := toJobDefinition(reg.Schedule)
	if err != nil {
		return nil, nil, nil, err
	}

	timeout := reg.Schedule.Timeout
	if timeout <= 0 {
		timeout = defaultJobTimeout
	}

	task := gocron.NewTask(func(ctx context.Context) error {
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return reg.Job.Run(runCtx)
	})

	jobOpts := []gocron.JobOption{
		gocron.WithName(name),
		gocron.WithEventListeners(
			gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
				logger.Debug("job starting", zap.String("job_name", jobName))
			}),
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				logger.Debug("job succeeded", zap.String("job_name", jobName))
			}),
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
				logger.Warn("job failed", zap.String("job_name", jobName), zap.Error(err))
			}),
			gocron.AfterJobRunsWithPanic(func(jobID uuid.UUID, jobName string, recoverData any) {
				logger.Error("job panic event", zap.String("job_name", jobName), zap.Any("panic", recoverData))
			}),
		),
	}

	if len(reg.Schedule.Tags) > 0 {
		jobOpts = append(jobOpts, gocron.WithTags(reg.Schedule.Tags...))
	}
	if reg.Schedule.Singleton {
		jobOpts = append(jobOpts, gocron.WithSingletonMode(gocron.LimitModeReschedule))
	}
	return jobDef, task, jobOpts, nil
}

func toJobDefinition(spec ScheduleSpec) (gocron.JobDefinition, error) {
	switch spec.Kind {
	case ScheduleCron:
		return gocron.CronJob(spec.Cron, spec.WithSeconds), nil
	case ScheduleDuration:
		return gocron.DurationJob(spec.Duration), nil
	default:
		return nil, fmt.Errorf("unsupported schedule kind: %s", spec.Kind)
	}
}
