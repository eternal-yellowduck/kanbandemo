package domain

import (
	"fmt"
	"strings"
	"time"
)

type RunKind string

const (
	RunDelivery RunKind = "delivery"
	RunCheck    RunKind = "check"
)

type RunStatus string

const (
	RunQueued        RunStatus = "queued"
	RunDispatched    RunStatus = "dispatched"
	RunRunning       RunStatus = "running"
	RunAwaitingInput RunStatus = "awaiting_input"
	RunCompleted     RunStatus = "completed"
	RunFailed        RunStatus = "failed"
	RunCancelled     RunStatus = "cancelled"
)

type RunFailure struct {
	Code    string
	Message string
}

type Run struct {
	ID               string
	TaskID           string
	CheckID          *string
	Kind             RunKind
	Status           RunStatus
	CurrentAttemptID *string
	Result           *DeliveryResult
	Failure          *RunFailure
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewRun(id, taskID string, kind RunKind, checkID *string, now time.Time) (Run, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(taskID) == "" || now.IsZero() {
		return Run{}, fmt.Errorf("%w: run id, task id and creation time are required", ErrValidation)
	}
	if kind != RunDelivery && kind != RunCheck {
		return Run{}, fmt.Errorf("%w: unsupported run kind %q", ErrValidation, kind)
	}
	if kind == RunCheck && (checkID == nil || strings.TrimSpace(*checkID) == "") {
		return Run{}, fmt.Errorf("%w: check run requires check id", ErrValidation)
	}
	if kind == RunDelivery && checkID != nil {
		return Run{}, fmt.Errorf("%w: delivery run cannot reference a check", ErrValidation)
	}
	return Run{ID: id, TaskID: taskID, CheckID: checkID, Kind: kind, Status: RunQueued, CreatedAt: now, UpdatedAt: now}, nil
}

func (r *Run) Dispatch(meta ChangeMeta) (DomainEvent, error) {
	return r.transition(RunQueued, RunDispatched, meta, EventRunDispatched, nil)
}

func (r *Run) Start(attemptID string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(attemptID) == "" {
		return DomainEvent{}, fmt.Errorf("%w: attempt id is required", ErrValidation)
	}
	event, err := r.transition(RunDispatched, RunRunning, meta, EventRunStarted, map[string]string{"attemptId": attemptID})
	if err == nil {
		r.CurrentAttemptID = &attemptID
	}
	return event, err
}

func (r *Run) RequestInput(blockID string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(blockID) == "" {
		return DomainEvent{}, fmt.Errorf("%w: block id is required", ErrValidation)
	}
	return r.transition(RunRunning, RunAwaitingInput, meta, EventInputRequested, map[string]string{"blockId": blockID})
}

func (r *Run) Resume(meta ChangeMeta) (DomainEvent, error) {
	return r.transition(RunAwaitingInput, RunQueued, meta, EventInputAnswered, nil)
}

func (r *Run) Complete(result DeliveryResult, meta ChangeMeta) (DomainEvent, error) {
	if err := result.validate(); err != nil {
		return DomainEvent{}, err
	}
	event, err := r.transition(RunRunning, RunCompleted, meta, EventRunCompleted, result)
	if err == nil {
		copy := result
		copy.Artifacts = append([]string(nil), result.Artifacts...)
		r.Result = &copy
	}
	return event, err
}

func (r *Run) Fail(failure RunFailure, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(failure.Code) == "" || strings.TrimSpace(failure.Message) == "" {
		return DomainEvent{}, fmt.Errorf("%w: failure code and message are required", ErrValidation)
	}
	event, err := r.transition(RunRunning, RunFailed, meta, EventRunFailed, failure)
	if err == nil {
		r.Failure = &failure
	}
	return event, err
}

func (r *Run) Cancel(meta ChangeMeta) (DomainEvent, error) {
	if r.Status != RunQueued && r.Status != RunDispatched && r.Status != RunRunning {
		return DomainEvent{}, fmt.Errorf("%w: run %s cannot be cancelled from %s", ErrInvalidTransition, r.ID, r.Status)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	r.Status = RunCancelled
	r.UpdatedAt = meta.OccurredAt
	return newEvent(meta, EventRunCancelled, "run", r.ID, nil)
}

func (r *Run) transition(from, to RunStatus, meta ChangeMeta, eventType string, payload any) (DomainEvent, error) {
	if r.Status != from {
		return DomainEvent{}, fmt.Errorf("%w: run %s cannot move from %s to %s", ErrInvalidTransition, r.ID, r.Status, to)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	r.Status = to
	r.UpdatedAt = meta.OccurredAt
	return newEvent(meta, eventType, "run", r.ID, payload)
}

type AttemptStatus string

const (
	AttemptRunning   AttemptStatus = "running"
	AttemptCompleted AttemptStatus = "completed"
	AttemptFailed    AttemptStatus = "failed"
	AttemptCancelled AttemptStatus = "cancelled"
)

type Attempt struct {
	ID         string
	RunID      string
	Number     int
	Harness    string
	Status     AttemptStatus
	SessionRef *string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func NewAttempt(id, runID string, number int, harness string) (Attempt, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(runID) == "" || number < 1 || strings.TrimSpace(harness) == "" {
		return Attempt{}, fmt.Errorf("%w: attempt identity, number and harness are required", ErrValidation)
	}
	return Attempt{ID: id, RunID: runID, Number: number, Harness: harness, Status: AttemptRunning}, nil
}

func (a *Attempt) Complete(meta ChangeMeta) (DomainEvent, error) {
	return a.finish(AttemptCompleted, EventRunCompleted, meta, nil)
}

func (a *Attempt) Fail(failure RunFailure, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(failure.Code) == "" || strings.TrimSpace(failure.Message) == "" {
		return DomainEvent{}, fmt.Errorf("%w: failure code and message are required", ErrValidation)
	}
	return a.finish(AttemptFailed, EventRunFailed, meta, failure)
}

func (a *Attempt) Cancel(meta ChangeMeta) (DomainEvent, error) {
	return a.finish(AttemptCancelled, EventRunCancelled, meta, nil)
}

func (a *Attempt) finish(status AttemptStatus, eventType string, meta ChangeMeta, payload any) (DomainEvent, error) {
	if a.Status != AttemptRunning {
		return DomainEvent{}, fmt.Errorf("%w: attempt %s cannot finish from %s", ErrInvalidTransition, a.ID, a.Status)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	a.Status = status
	a.FinishedAt = &meta.OccurredAt
	return newEvent(meta, eventType, "attempt", a.ID, payload)
}
