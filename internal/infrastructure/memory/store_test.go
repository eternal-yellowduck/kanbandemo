package memory

import (
	"context"
	"testing"
	"time"

	"github.com/eternal-yellowduck/kanbandemo/internal/domain"
)

func TestStoreCopiesMutableValuesAndDeduplicatesEvents(t *testing.T) {
	store := NewStore()
	task, err := domain.NewTask("task-1", "小猫网站", "描述", []string{"标准"}, memoryTestTime(0))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	task.AcceptanceCriteria[0] = "外部修改"
	loaded, ok, err := store.GetTask(context.Background(), task.ID)
	if err != nil || !ok {
		t.Fatalf("get task = %v, %v", loaded, err)
	}
	if loaded.AcceptanceCriteria[0] != "标准" {
		t.Fatalf("store leaked mutable slice: %v", loaded.AcceptanceCriteria)
	}

	event := domain.DomainEvent{ID: "event-1", Type: domain.EventTaskCreated, AggregateType: "task", AggregateID: task.ID, OccurredAt: memoryTestTime(1), Source: "test", Reason: "create", IdempotencyKey: "idem-1"}
	if err := store.AppendEvents(context.Background(), event, event); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListEvents(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
}

func memoryTestTime(offset int) time.Time {
	return time.Date(2026, 8, 28, 0, 0, offset, 0, time.UTC)
}
