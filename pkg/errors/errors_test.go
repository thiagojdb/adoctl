package errors

import (
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "basic error without underlying",
			err:      &Error{Code: ExitCodeGeneral, Message: "test error"},
			expected: "test error",
		},
		{
			name:     "error with underlying",
			err:      &Error{Code: ExitCodeConfig, Message: "config error", Underlying: errors.New("file not found")},
			expected: "config error: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &Error{
		Code:       ExitCodeGeneral,
		Message:    "test error",
		Underlying: underlying,
	}

	if err.Unwrap() != underlying {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), underlying)
	}
}

func TestNew(t *testing.T) {
	err := New(ExitCodeConfig, "configuration error")

	if err.Code != ExitCodeConfig {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeConfig)
	}
	if err.Message != "configuration error" {
		t.Errorf("Message = %q, want %q", err.Message, "configuration error")
	}
	if err.Underlying != nil {
		t.Errorf("Underlying = %v, want nil", err.Underlying)
	}
}

func TestNewWithError(t *testing.T) {
	underlying := errors.New("API error")
	err := NewWithError(ExitCodeAPIAuth, "authentication failed", underlying)

	if err.Code != ExitCodeAPIAuth {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeAPIAuth)
	}
	if err.Message != "authentication failed" {
		t.Errorf("Message = %q, want %q", err.Message, "authentication failed")
	}
	if err.Underlying != underlying {
		t.Errorf("Underlying = %v, want %v", err.Underlying, underlying)
	}
}

func TestNewWithSuggestion(t *testing.T) {
	err := NewWithSuggestion(ExitCodeValidation, "invalid input", "Check the documentation for valid values")

	if err.Code != ExitCodeValidation {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeValidation)
	}
	if err.Message != "invalid input" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid input")
	}
	if err.Suggestion != "Check the documentation for valid values" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "Check the documentation for valid values")
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("original error")
	err := Wrap(underlying, "wrapped message")

	if err.Error() != "wrapped message: original error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "wrapped message: original error")
	}

	if Wrap(nil, "message") != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestWrapWithCode(t *testing.T) {
	underlying := errors.New("original error")
	err := WrapWithCode(underlying, ExitCodeNotImplemented, "not implemented")

	if err.Code != ExitCodeNotImplemented {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeNotImplemented)
	}
	if err.Message != "not implemented: original error" {
		t.Errorf("Message = %q, want %q", err.Message, "not implemented: original error")
	}
}

func TestWrapWrapsError(t *testing.T) {
	wrappedErr := New(ExitCodeAPINotFound, "not found error")
	err := Wrap(wrappedErr, "outer error")

	if err.Code != ExitCodeAPINotFound {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeAPINotFound)
	}
	if err.Message != "outer error: not found error" {
		t.Errorf("Message = %q, want %q", err.Message, "outer error: not found error")
	}
}

func TestIs(t *testing.T) {
	err1 := New(ExitCodeConfig, "error 1")
	err2 := New(ExitCodeConfig, "error 2")
	err3 := New(ExitCodeGeneral, "error 3")

	if !Is(err1, err2) {
		t.Error("Is() should return true for same exit code")
	}

	if Is(err1, err3) {
		t.Error("Is() should return false for different exit codes")
	}

	if Is(err1, errors.New("plain error")) {
		t.Error("Is() should return false for plain error")
	}
}

func TestIsExitCode(t *testing.T) {
	err := New(ExitCodeAPIAuth, "auth error")

	if !IsExitCode(err, ExitCodeAPIAuth) {
		t.Error("IsExitCode() should return true for matching code")
	}

	if IsExitCode(err, ExitCodeConfig) {
		t.Error("IsExitCode() should return false for non-matching code")
	}

	if IsExitCode(nil, ExitCodeGeneral) {
		t.Error("IsExitCode() should return false for nil error")
	}

	if IsExitCode(errors.New("plain error"), ExitCodeGeneral) {
		t.Error("IsExitCode() should return false for plain error")
	}
}

func TestHandle(t *testing.T) {
	t.Run("nil error does nothing", func(t *testing.T) {
		Handle(nil)
	})

	t.Run("structured error does not panic", func(t *testing.T) {
		err := NewWithSuggestion(ExitCodeConfig, "configuration missing", "Run adoctl setup to configure")
		if err == nil {
			t.Error("Expected non-nil error")
		}
	})
}

func TestHandleQuiet(t *testing.T) {
	t.Run("nil error does nothing", func(t *testing.T) {
		HandleQuiet(nil)
	})

	t.Run("plain error does not panic", func(t *testing.T) {
		err := errors.New("plain error")
		if err == nil {
			t.Error("Expected non-nil error")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name  string
		fn    func() *Error
		check func(*Error) bool
	}{
		{
			name:  "APIError",
			fn:    func() *Error { return APIError(errors.New("timeout")) },
			check: func(e *Error) bool { return e.Code == ExitCodeAPIRequest },
		},
		{
			name:  "AuthError",
			fn:    func() *Error { return AuthError() },
			check: func(e *Error) bool { return e.Code == ExitCodeAPIAuth },
		},
		{
			name:  "NotFoundError",
			fn:    func() *Error { return NotFoundError("repository") },
			check: func(e *Error) bool { return e.Code == ExitCodeAPINotFound },
		},
		{
			name:  "ConfigError",
			fn:    func() *Error { return ConfigError("invalid yaml") },
			check: func(e *Error) bool { return e.Code == ExitCodeConfig },
		},
		{
			name:  "ValidationError",
			fn:    func() *Error { return ValidationError("missing required field") },
			check: func(e *Error) bool { return e.Code == ExitCodeValidation },
		},
		{
			name:  "TimeoutError",
			fn:    func() *Error { return TimeoutError("API call") },
			check: func(e *Error) bool { return e.Code == ExitCodeTimeout },
		},
		{
			name:  "CancelledError",
			fn:    func() *Error { return CancelledError("user cancelled") },
			check: func(e *Error) bool { return e.Code == ExitCodeCancellation },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !tt.check(err) {
				t.Errorf("%s() returned error with unexpected code %d", tt.name, err.Code)
			}
		})
	}
}

func TestUserError(t *testing.T) {
	err := UserError(ExitCodeValidation, "invalid input")

	if err.Code != ExitCodeValidation {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeValidation)
	}
	if err.Message != "invalid input" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid input")
	}
}

func TestUserErrorWithSuggestion(t *testing.T) {
	err := UserErrorWithSuggestion(ExitCodeNotImplemented, "feature not ready", "Try again later")

	if err.Code != ExitCodeNotImplemented {
		t.Errorf("Code = %d, want %d", err.Code, ExitCodeNotImplemented)
	}
	if err.Message != "feature not ready" {
		t.Errorf("Message = %q, want %q", err.Message, "feature not ready")
	}
	if err.Suggestion != "Try again later" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "Try again later")
	}
}
