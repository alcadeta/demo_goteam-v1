package task

import (
	"github.com/kxplxn/goteam/pkg/validator"
)

// TitleValidator can be used to validate a task title.
type TitleValidator struct{}

// NewTitleValidator creates and returns a new TitleValidator.
func NewTitleValidator() TitleValidator { return TitleValidator{} }

// Validate validates a given task title.
func (v TitleValidator) Validate(title string) error {
	if title == "" {
		return validator.ErrEmpty
	}
	if len(title) > 50 {
		return validator.ErrTooLong
	}
	return nil
}

// ColNoValidator can be used to validate a task's column number.
type ColNoValidator struct{}

// NewColNoValidator creates and returns a new ColNoValidator.
func NewColNoValidator() ColNoValidator { return ColNoValidator{} }

// Validate validates a task's column number.
func (v ColNoValidator) Validate(number int) error {
	if number < 0 || number > 3 {
		return validator.ErrOutOfBounds
	}
	return nil
}
