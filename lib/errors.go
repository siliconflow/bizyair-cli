package lib

import (
	"errors"
	"fmt"
)

// StepError 包含步骤信息的错误类型
type StepError struct {
	Step string
	Err  error
}

func (e *StepError) Error() string {
	if e.Step != "" {
		return fmt.Sprintf("[%s] %v", e.Step, e.Err)
	}
	return e.Err.Error()
}

func (e *StepError) Unwrap() error {
	return e.Err
}

// WithStep 为错误添加步骤信息
func WithStep(step string, err error) error {
	if err == nil {
		return nil
	}
	return &StepError{
		Step: step,
		Err:  err,
	}
}

// GetStep 从错误中提取步骤信息
func GetStep(err error) string {
	if err == nil {
		return ""
	}
	var stepErr *StepError
	if errors.As(err, &stepErr) {
		return stepErr.Step
	}
	return ""
}
