package cron

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type panicJob struct{}

func (j *panicJob) Name() string { return "panic-job" }

func (j *panicJob) Run(context.Context) error {
	panic("boom")
}

type blockingJob struct {
	started chan struct{}
	stopped chan struct{}
}

func (j *blockingJob) Name() string { return "blocking-job" }

func (j *blockingJob) Run(ctx context.Context) error {
	close(j.started)
	<-ctx.Done()
	close(j.stopped)
	return nil
}

func TestBuildGocronJobPropagatesPanicsToGocron(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	reg := JobRegistration{
		Job: &panicJob{},
		Schedule: ScheduleSpec{
			Kind:      ScheduleDuration,
			Duration:  10 * time.Millisecond,
			Timeout:   time.Second,
			Singleton: true,
		},
	}

	jobDef, task, jobOpts, err := buildGocronJob(reg, logger)
	if err != nil {
		t.Fatalf("build job: %v", err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	defer func() { _ = s.Shutdown() }()

	if _, err := s.NewJob(jobDef, task, jobOpts...); err != nil {
		t.Fatalf("register job: %v", err)
	}

	s.Start()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hasLogMessage(recorded.AllUntimed(), "job panic event") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected panic event log, got logs: %+v", recorded.AllUntimed())
}

func TestBuildGocronJobCancelsContextOnShutdown(t *testing.T) {
	t.Parallel()

	job := &blockingJob{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}

	reg := JobRegistration{
		Job: job,
		Schedule: ScheduleSpec{
			Kind:     ScheduleDuration,
			Duration: 10 * time.Millisecond,
			Timeout:  5 * time.Second,
		},
	}

	jobDef, task, jobOpts, err := buildGocronJob(reg, zap.NewNop())
	if err != nil {
		t.Fatalf("build job: %v", err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	defer func() { _ = s.Shutdown() }()

	if _, err := s.NewJob(jobDef, task, jobOpts...); err != nil {
		t.Fatalf("register job: %v", err)
	}

	s.Start()

	select {
	case <-job.started:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not start")
	}

	if err := s.Shutdown(); err != nil {
		t.Fatalf("shutdown scheduler: %v", err)
	}

	select {
	case <-job.stopped:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected job context to be canceled on shutdown")
	}
}

func hasLogMessage(entries []observer.LoggedEntry, message string) bool {
	for _, entry := range entries {
		if entry.Message == message || strings.Contains(entry.Message, message) {
			return true
		}
	}
	return false
}
