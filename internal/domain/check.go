package domain

import (
	"fmt"
	"strings"
	"time"
)

type CheckKind string

const (
	CheckAutomatedTest CheckKind = "automated_test"
	CheckCodeReview    CheckKind = "code_review"
)

type CheckStatus string

const (
	CheckPending          CheckStatus = "pending"
	CheckRunning          CheckStatus = "running"
	CheckPassed           CheckStatus = "passed"
	CheckChangesRequested CheckStatus = "changes_requested"
	CheckFailed           CheckStatus = "failed"
	CheckUnableToRun      CheckStatus = "unable_to_run"
	CheckCancelled        CheckStatus = "cancelled"
)

type Check struct {
	ID            string
	TaskID        string
	Kind          CheckKind
	Required      bool
	Revision      int
	Status        CheckStatus
	AssignedRunID *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewCheck(id, taskID string, kind CheckKind, required bool, revision int, now time.Time) (Check, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(taskID) == "" || now.IsZero() || revision < 1 {
		return Check{}, fmt.Errorf("%w: check identity, revision and creation time are required", ErrValidation)
	}
	if kind != CheckAutomatedTest && kind != CheckCodeReview {
		return Check{}, fmt.Errorf("%w: unsupported check kind %q", ErrValidation, kind)
	}
	return Check{ID: id, TaskID: taskID, Kind: kind, Required: required, Revision: revision, Status: CheckPending, CreatedAt: now, UpdatedAt: now}, nil
}

func (c *Check) AssignRun(runID string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(runID) == "" {
		return DomainEvent{}, fmt.Errorf("%w: run id is required", ErrValidation)
	}
	if c.Status != CheckPending {
		return DomainEvent{}, fmt.Errorf("%w: check %s can only be assigned while pending", ErrInvalidTransition, c.ID)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	c.AssignedRunID = &runID
	c.Status = CheckRunning
	c.UpdatedAt = meta.OccurredAt
	return newEvent(meta, EventCheckStarted, "check", c.ID, map[string]string{"runId": runID})
}

func (c *Check) Pass(meta ChangeMeta) (DomainEvent, error) {
	return c.finish(CheckPassed, EventCheckPassed, meta, nil)
}

func (c *Check) RequestChanges(meta ChangeMeta) (DomainEvent, error) {
	return c.finish(CheckChangesRequested, EventChangesRequested, meta, nil)
}

func (c *Check) Fail(reason string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(reason) == "" {
		return DomainEvent{}, fmt.Errorf("%w: check failure reason is required", ErrValidation)
	}
	return c.finish(CheckFailed, EventCheckFailed, meta, map[string]string{"reason": reason})
}

func (c *Check) UnableToRun(reason string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(reason) == "" {
		return DomainEvent{}, fmt.Errorf("%w: check failure reason is required", ErrValidation)
	}
	return c.finish(CheckUnableToRun, EventCheckFailed, meta, map[string]string{"reason": reason})
}

func (c *Check) Cancel(meta ChangeMeta) (DomainEvent, error) {
	return c.finish(CheckCancelled, EventRunCancelled, meta, nil)
}

func (c *Check) finish(status CheckStatus, eventType string, meta ChangeMeta, payload any) (DomainEvent, error) {
	if c.Status != CheckRunning {
		return DomainEvent{}, fmt.Errorf("%w: check %s cannot finish from %s", ErrInvalidTransition, c.ID, c.Status)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	c.Status = status
	c.UpdatedAt = meta.OccurredAt
	return newEvent(meta, eventType, "check", c.ID, payload)
}
