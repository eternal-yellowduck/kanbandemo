package domain

import "errors"

var (
	ErrValidation         = errors.New("validation error")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrInvariantViolation = errors.New("domain invariant violation")
)
