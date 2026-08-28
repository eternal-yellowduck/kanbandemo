package domain

import (
	"errors"
	"testing"
	"time"
)

func taskForTest(t *testing.T) Task {
	t.Helper()
	task, err := NewTask("task-1", "小猫网站", "做一个网站", []string{"页面可访问"}, testTime(0))
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func metaForTest(index int) ChangeMeta {
	return ChangeMeta{
		EventID:        "event-" + string(rune('a'+index)),
		Source:         "test",
		Reason:         "test_transition",
		IdempotencyKey: "idem-" + string(rune('a'+index)),
		OccurredAt:     testTime(index + 1),
	}
}

func testTime(offset int) time.Time {
	return time.Date(2026, 8, 28, 0, 0, offset, 0, time.UTC)
}

func TestTaskLifecycleRequiresReviewAndAcceptance(t *testing.T) {
	task := taskForTest(t)
	if _, err := task.Start(metaForTest(0)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("start from backlog error = %v, want invalid transition", err)
	}
	if _, err := task.MoveToTodo(metaForTest(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := task.Start(metaForTest(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := task.Accept(true, metaForTest(2)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("accept from in_progress error = %v, want invalid transition", err)
	}
	result := DeliveryResult{Summary: "已完成网站", Artifacts: []string{"index.html"}}
	if _, err := task.MarkDeliveryCompleted(result, metaForTest(2)); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskReview {
		t.Fatalf("status = %s, want review", task.Status)
	}
	if _, err := task.Accept(false, metaForTest(3)); !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("accept without checks error = %v, want invariant violation", err)
	}
	if _, err := task.Accept(true, metaForTest(4)); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskDone {
		t.Fatalf("status = %s, want done", task.Status)
	}
}

func TestTaskChangesRequestedReturnsToInProgress(t *testing.T) {
	task := taskForTest(t)
	_, _ = task.MoveToTodo(metaForTest(0))
	_, _ = task.Start(metaForTest(1))
	_, _ = task.MarkDeliveryCompleted(DeliveryResult{Summary: "交付"}, metaForTest(2))
	if _, err := task.RequestChanges(metaForTest(3)); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskInProgress {
		t.Fatalf("status = %s, want in_progress", task.Status)
	}
}

func TestTaskBlockDoesNotChangeTopLevelStatus(t *testing.T) {
	task := taskForTest(t)
	event, err := task.SetBlock("block-1", metaForTest(0))
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != EventInputRequested || task.Status != TaskBacklog {
		t.Fatalf("block event/status = %s/%s", event.Type, task.Status)
	}
	if task.ActiveBlockID == nil || *task.ActiveBlockID != "block-1" {
		t.Fatal("active block was not recorded")
	}
	if _, err := task.ClearBlock(metaForTest(1)); err != nil {
		t.Fatal(err)
	}
	if task.ActiveBlockID != nil {
		t.Fatal("active block was not cleared")
	}
}
