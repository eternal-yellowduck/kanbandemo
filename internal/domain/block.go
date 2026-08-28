package domain

import (
	"fmt"
	"strings"
	"time"
)

type BlockKind string

const BlockHumanInput BlockKind = "human_input"

type Block struct {
	ID         string
	TaskID     string
	RunID      string
	Kind       BlockKind
	Question   string
	Options    []string
	Answer     *string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

func NewHumanInputBlock(id, taskID, runID, question string, options []string, now time.Time) (Block, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(taskID) == "" || strings.TrimSpace(runID) == "" || strings.TrimSpace(question) == "" || now.IsZero() {
		return Block{}, fmt.Errorf("%w: block identity, question and creation time are required", ErrValidation)
	}
	return Block{ID: id, TaskID: taskID, RunID: runID, Kind: BlockHumanInput, Question: question, Options: append([]string(nil), options...), CreatedAt: now}, nil
}

func (b *Block) SubmitAnswer(answer string, meta ChangeMeta) (DomainEvent, error) {
	if strings.TrimSpace(answer) == "" {
		return DomainEvent{}, fmt.Errorf("%w: answer is required", ErrValidation)
	}
	if b.Answer != nil {
		return DomainEvent{}, fmt.Errorf("%w: block %s already has an answer", ErrInvalidTransition, b.ID)
	}
	if err := meta.validate(); err != nil {
		return DomainEvent{}, err
	}
	value := strings.TrimSpace(answer)
	b.Answer = &value
	b.ResolvedAt = &meta.OccurredAt
	return newEvent(meta, EventInputAnswered, "block", b.ID, map[string]string{"answer": value})
}
