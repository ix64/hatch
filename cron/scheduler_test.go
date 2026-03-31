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

func hasLogMessage(entries []observer.LoggedEntry, message string) bool {
	for _, entry := range entries {
		if entry.Message == message || strings.Contains(entry.Message, message) {
			return true
		}
	}
	return false
}
