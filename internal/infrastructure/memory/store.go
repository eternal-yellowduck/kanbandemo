package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/eternal-yellowduck/kanbandemo/internal/domain"
)

type Store struct {
	mu       sync.RWMutex
	tasks    map[string]domain.Task
	runs     map[string]domain.Run
	attempts map[string]domain.Attempt
	checks   map[string]domain.Check
	blocks   map[string]domain.Block
	events   []domain.DomainEvent
	eventIDs map[string]struct{}
	idemKeys map[string]struct{}
}

func NewStore() *Store {
	return &Store{
		tasks:    make(map[string]domain.Task),
		runs:     make(map[string]domain.Run),
		attempts: make(map[string]domain.Attempt),
		checks:   make(map[string]domain.Check),
		blocks:   make(map[string]domain.Block),
		eventIDs: make(map[string]struct{}),
		idemKeys: make(map[string]struct{}),
	}
}

func (s *Store) SaveTask(_ context.Context, task domain.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task.AcceptanceCriteria = append([]string(nil), task.AcceptanceCriteria...)
	s.tasks[task.ID] = task
	return nil
}

func (s *Store) GetTask(_ context.Context, id string) (domain.Task, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if ok {
		task.AcceptanceCriteria = append([]string(nil), task.AcceptanceCriteria...)
	}
	return task, ok, nil
}

func (s *Store) SaveRun(_ context.Context, run domain.Run) error {
	if run.ID == "" {
		return fmt.Errorf("run id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.CheckID != nil {
		value := *run.CheckID
		run.CheckID = &value
	}
	if run.CurrentAttemptID != nil {
		value := *run.CurrentAttemptID
		run.CurrentAttemptID = &value
	}
	if run.Result != nil {
		result := *run.Result
		result.Artifacts = append([]string(nil), result.Artifacts...)
		run.Result = &result
	}
	if run.Failure != nil {
		failure := *run.Failure
		run.Failure = &failure
	}
	s.runs[run.ID] = run
	return nil
}

func (s *Store) GetRun(_ context.Context, id string) (domain.Run, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	return cloneRun(run), ok, nil
}

func (s *Store) SaveAttempt(_ context.Context, attempt domain.Attempt) error {
	if attempt.ID == "" {
		return fmt.Errorf("attempt id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if attempt.SessionRef != nil {
		value := *attempt.SessionRef
		attempt.SessionRef = &value
	}
	s.attempts[attempt.ID] = attempt
	return nil
}

func (s *Store) GetAttempt(_ context.Context, id string) (domain.Attempt, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	attempt, ok := s.attempts[id]
	if attempt.SessionRef != nil {
		value := *attempt.SessionRef
		attempt.SessionRef = &value
	}
	return attempt, ok, nil
}

func (s *Store) SaveCheck(_ context.Context, check domain.Check) error {
	if check.ID == "" {
		return fmt.Errorf("check id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if check.AssignedRunID != nil {
		value := *check.AssignedRunID
		check.AssignedRunID = &value
	}
	s.checks[check.ID] = check
	return nil
}

func (s *Store) GetCheck(_ context.Context, id string) (domain.Check, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	check, ok := s.checks[id]
	if check.AssignedRunID != nil {
		value := *check.AssignedRunID
		check.AssignedRunID = &value
	}
	return check, ok, nil
}

func (s *Store) SaveBlock(_ context.Context, block domain.Block) error {
	if block.ID == "" {
		return fmt.Errorf("block id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	block.Options = append([]string(nil), block.Options...)
	if block.Answer != nil {
		value := *block.Answer
		block.Answer = &value
	}
	s.blocks[block.ID] = block
	return nil
}

func (s *Store) GetBlock(_ context.Context, id string) (domain.Block, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	block, ok := s.blocks[id]
	block.Options = append([]string(nil), block.Options...)
	if block.Answer != nil {
		value := *block.Answer
		block.Answer = &value
	}
	return block, ok, nil
}

func (s *Store) AppendEvents(_ context.Context, events ...domain.DomainEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, event := range events {
		if event.ID == "" || event.IdempotencyKey == "" || event.Type == "" || event.AggregateID == "" {
			return fmt.Errorf("event identity and idempotency key are required")
		}
		if _, seen := s.eventIDs[event.ID]; seen {
			continue
		}
		if _, seen := s.idemKeys[event.IdempotencyKey]; seen {
			continue
		}
		s.events = append(s.events, event)
		s.eventIDs[event.ID] = struct{}{}
		s.idemKeys[event.IdempotencyKey] = struct{}{}
	}
	return nil
}

func (s *Store) ListEvents(_ context.Context, aggregateID string) ([]domain.DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.DomainEvent, 0)
	for _, event := range s.events {
		if event.AggregateID == aggregateID {
			result = append(result, event)
		}
	}
	return result, nil
}

func cloneRun(run domain.Run) domain.Run {
	if run.CheckID != nil {
		value := *run.CheckID
		run.CheckID = &value
	}
	if run.CurrentAttemptID != nil {
		value := *run.CurrentAttemptID
		run.CurrentAttemptID = &value
	}
	if run.Result != nil {
		result := *run.Result
		result.Artifacts = append([]string(nil), result.Artifacts...)
		run.Result = &result
	}
	if run.Failure != nil {
		failure := *run.Failure
		run.Failure = &failure
	}
	return run
}
