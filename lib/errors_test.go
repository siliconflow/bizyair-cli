package lib

import (
	"errors"
	"testing"
)

func TestStepError_Error(t *testing.T) {
	tests := []struct {
		name string
		step string
		err  error
		want string
	}{
		{"with step", "上传", errors.New("network error"), "[上传] network error"},
		{"empty step", "", errors.New("network error"), "network error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &StepError{Step: tt.step, Err: tt.err}
			got := se.Error()
			if got != tt.want {
				t.Errorf("StepError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStepError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	se := &StepError{Step: "test", Err: inner}
	if !errors.Is(se, inner) {
		t.Error("StepError should unwrap to inner error")
	}
}

func TestWithStep(t *testing.T) {
	tests := []struct {
		name    string
		step    string
		err     error
		wantNil bool
	}{
		{"wraps error", "上传", errors.New("fail"), false},
		{"nil error returns nil", "上传", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithStep(tt.step, tt.err)
			if tt.wantNil {
				if got != nil {
					t.Error("WithStep with nil error should return nil")
				}
				return
			}
			if got == nil {
				t.Fatal("WithStep should not return nil for non-nil error")
			}
			var se *StepError
			if !errors.As(got, &se) {
				t.Error("result should be a StepError")
			}
			if se.Step != tt.step {
				t.Errorf("Step = %q, want %q", se.Step, tt.step)
			}
		})
	}
}

func TestGetStep(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"extract step from StepError", WithStep("检查模型", errors.New("fail")), "检查模型"},
		{"returns empty for plain error", errors.New("plain error"), ""},
		{"returns empty for nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStep(tt.err)
			if got != tt.want {
				t.Errorf("GetStep() = %q, want %q", got, tt.want)
			}
		})
	}
}
