package domain

import (
	"errors"
	"testing"
)

func TestRunRequiresStructuredCompletion(t *testing.T) {
	run, err := NewRun("run-1", "task-1", RunDelivery, nil, testTime(0))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = run.Dispatch(metaForTest(0))
	if _, err := run.Start("attempt-1", metaForTest(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Complete(DeliveryResult{}, metaForTest(2)); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty result error = %v, want validation", err)
	}
	if _, err := run.Complete(DeliveryResult{Summary: "完成"}, metaForTest(2)); err != nil {
		t.Fatal(err)
	}
	if run.Status != RunCompleted || run.Result == nil {
		t.Fatalf("run status/result = %s/%v", run.Status, run.Result)
	}
}

func TestRunInputPauseAndResume(t *testing.T) {
	run, err := NewRun("run-1", "task-1", RunDelivery, nil, testTime(0))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = run.Dispatch(metaForTest(0))
	_, _ = run.Start("attempt-1", metaForTest(1))
	if _, err := run.RequestInput("block-1", metaForTest(2)); err != nil {
		t.Fatal(err)
	}
	if run.Status != RunAwaitingInput {
		t.Fatalf("status = %s, want awaiting_input", run.Status)
	}
	if _, err := run.Resume(metaForTest(3)); err != nil {
		t.Fatal(err)
	}
	if run.Status != RunQueued {
		t.Fatalf("status = %s, want queued", run.Status)
	}
}

func TestRunCheckRequiresCheckID(t *testing.T) {
	if _, err := NewRun("run-1", "task-1", RunCheck, nil, testTime(0)); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing check id error = %v, want validation", err)
	}
}

func TestAttemptLifecycle(t *testing.T) {
	attempt, err := NewAttempt("attempt-1", "run-1", 1, "fake")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attempt.Complete(metaForTest(0)); err != nil {
		t.Fatal(err)
	}
	if attempt.Status != AttemptCompleted || attempt.FinishedAt == nil {
		t.Fatalf("status/finishedAt = %s/%v", attempt.Status, attempt.FinishedAt)
	}
	if _, err := attempt.Cancel(metaForTest(1)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cancel completed attempt error = %v, want invalid transition", err)
	}
}
