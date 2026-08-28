package domain

import (
	"fmt"
	"strings"
	"time"
)

type TaskStatus string

const (
	TaskBacklog    TaskStatus = "backlog"
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskDone       TaskStatus = "done"
)

type DeliveryResult struct {
	Summary   string
	Artifacts []string
	Handoff   string
}

func (r DeliveryResult) validate() error {
	if strings.TrimSpace(r.Summary) == "" {
		return fmt.Errorf("%w: delivery summary is required", ErrValidation)
	}
	return nil
}

type Task struct {
	ID                 string
	Title              string
	Description        string
	AcceptanceCriteria []string
	Status             TaskStatus
	ActiveBlockID      *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewTask(id, title, description string, acceptanceCriteria []string, now time.Time) (Task, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" || now.IsZero() {
		return Task{}, fmt.Errorf("%w: task id, title and creation time are required", ErrValidation)
	}
	return Task{
		ID:                 id,
		Title:              strings.TrimSpace(title),
		Description:        description,
		AcceptanceCriteria: append([]string(nil), acceptanceCriteria...),
		Status:             TaskBacklog,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func (t *Task) MoveToTodo(meta ChangeMeta) (DomainEvent, error) {
	return t.transition(TaskBacklog, TaskTodo, meta, "task_moved_to_todo", nil)
}

func (t *Task) Start(meta ChangeMeta) (DomainEvent, error) {
	return t.transition(TaskTodo, TaskInProgress, meta, EventTaskStarted, nil)
}

func (t *Task) MarkDeliveryCompleted(result DeliveryResult, meta ChangeMeta) (DomainEvent, error) {
	if err := result.validate(); err != nil {
		return DomainEvent{}, err
	}
	return t.transition(TaskInProgress, TaskReview, meta, EventTaskReview, result)
}

func (t *Task) RequestChanges(meta ChangeMeta) (DomainEvent, error) {
	return t.transition(TaskReview, TaskInProgress, meta, EventChangesRequested, nil)
}

func (t *Task) Accept(requiredChecksPassed bool, meta ChangeMeta) (DomainEvent, error) {
	if !requiredChecksPassed {
		return DomainEvent{}, fmt.Errorf("%w: all required checks must pass before acceptance", ErrInvariantViolation)
	}
	return t.transition(TaskReview, TaskDone, meta, EventTaskCompleted, nil)
}

func (t *Task) SetBlock(blockID string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(blockID) == "" {
		return DomainEvent{}, fmt.Errorf("%w: block id is required", ErrValidation)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	t.ActiveBlockID = &blockID
	t.UpdatedAt = meta.OccurredAt
	return newEvent(meta, EventInputRequested, "task", t.ID, map[string]string{"blockId": blockID})
}

func (t *Task) ClearBlock(meta ChangeMeta) (DomainEvent, error) {
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	t.ActiveBlockID = nil
	t.UpdatedAt = meta.OccurredAt
	return newEvent(meta, EventInputAnswered, "task", t.ID, nil)
}

func (t *Task) transition(from, to TaskStatus, meta ChangeMeta, eventType string, payload any) (DomainEvent, error) {
	if t.Status != from {
		return DomainEvent{}, fmt.Errorf("%w: task %s cannot move from %s to %s", ErrInvalidTransition, t.ID, t.Status, to)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	t.Status = to
	t.UpdatedAt = meta.OccurredAt
	return newEvent(meta, eventType, "task", t.ID, payload)
}
