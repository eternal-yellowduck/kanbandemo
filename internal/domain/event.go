package domain

import (
	"fmt"
	"time"
)

const (
	EventTaskCreated      = "task_created"
	EventTaskStarted      = "task_started"
	EventTaskReview       = "task_review_started"
	EventTaskCompleted    = "task_completed"
	EventRunQueued        = "run_queued"
	EventRunDispatched    = "run_dispatched"
	EventRunStarted       = "run_started"
	EventRunCompleted     = "run_completed"
	EventRunFailed        = "run_failed"
	EventRunCancelled     = "run_cancelled"
	EventInputRequested   = "input_requested"
	EventInputAnswered    = "input_answered"
	EventCheckCreated     = "check_created"
	EventCheckStarted     = "check_started"
	EventCheckPassed      = "check_passed"
	EventChangesRequested = "changes_requested"
	EventCheckFailed      = "check_failed"
)

type ChangeMeta struct {
	EventID        string
	Source         string
	Reason         string
	IdempotencyKey string
	OccurredAt     time.Time
}

func (m ChangeMeta) validate() error {
	if m.EventID == "" || m.Source == "" || m.Reason == "" || m.IdempotencyKey == "" || m.OccurredAt.IsZero() {
		return fmt.Errorf("%w: event metadata is incomplete", ErrValidation)
	}
	return nil
}

type DomainEvent struct {
	ID             string
	Type           string
	AggregateType  string
	AggregateID    string
	OccurredAt     time.Time
	Source         string
	Reason         string
	IdempotencyKey string
	Payload        any
}

func newEvent(meta ChangeMeta, eventType, aggregateType, aggregateID string, payload any) (DomainEvent, error) {
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	if eventType == "" || aggregateType == "" || aggregateID == "" {
		return DomainEvent{}, fmt.Errorf("%w: event identity is incomplete", ErrValidation)
	}
	return DomainEvent{
		ID:             meta.EventID,
		Type:           eventType,
		AggregateType:  aggregateType,
		AggregateID:    aggregateID,
		OccurredAt:     meta.OccurredAt,
		Source:         meta.Source,
		Reason:         meta.Reason,
		IdempotencyKey: meta.IdempotencyKey,
		Payload:        payload,
	}, nil
}
