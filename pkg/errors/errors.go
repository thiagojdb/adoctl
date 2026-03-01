package errors

import (
	"fmt"
	"os"
	"strings"

	"adoctl/pkg/logger"

	"github.com/fatih/color"
)

type ExitCode int

const (
	ExitCodeSuccess        ExitCode = 0
	ExitCodeGeneral        ExitCode = 1
	ExitCodeConfig         ExitCode = 2
	ExitCodeAPIAuth        ExitCode = 3
	ExitCodeAPINotFound    ExitCode = 4
	ExitCodeAPIRequest     ExitCode = 5
	ExitCodeValidation     ExitCode = 6
	ExitCodeFileOperation  ExitCode = 7
	ExitCodeCancellation   ExitCode = 8
	ExitCodeTimeout        ExitCode = 9
	ExitCodeNotImplemented ExitCode = 10
)

// Standardized error messages for consistent user-facing errors
const (
	ErrMsgServiceCreation  = "Failed to initialize Azure DevOps service"
	ErrMsgRepoNotFound     = "Could not determine repository"
	ErrMsgBranchNotFound   = "Could not determine branch"
	ErrMsgPRCreateFailed   = "Failed to create pull request"
	ErrMsgPRListFailed     = "Failed to list pull requests"
	ErrMsgPRUpdateFailed   = "Failed to update pull request"
	ErrMsgPRMergeFailed    = "Failed to merge pull request"
	ErrMsgBuildListFailed  = "Failed to list builds"
	ErrMsgDeployListFailed = "Failed to list deployments"
	ErrMsgCacheFailed      = "Cache operation failed"
	ErrMsgInvalidInput     = "Invalid input provided"
)

type Error struct {
	Code       ExitCode
	Message    string
	Underlying error
	Suggestion string
}

func (e *Error) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Underlying)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Underlying
}

func New(code ExitCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func NewWithError(code ExitCode, message string, err error) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Underlying: err,
	}
}

func NewWithSuggestion(code ExitCode, message string, suggestion string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
	}
}

func NewWithAll(code ExitCode, message string, err error, suggestion string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Underlying: err,
		Suggestion: suggestion,
	}
}

func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}

	if wrapped, ok := err.(*Error); ok {
		return &Error{
			Code:       wrapped.Code,
			Message:    message + ": " + wrapped.Message,
			Underlying: wrapped.Underlying,
			Suggestion: wrapped.Suggestion,
		}
	}

	return &Error{
		Code:       ExitCodeGeneral,
		Message:    message,
		Underlying: err,
	}
}

func WrapWithCode(err error, code ExitCode, message string) *Error {
	if err == nil {
		return nil
	}

	var errMsg string
	if wrapped, ok := err.(*Error); ok {
		errMsg = wrapped.Message
		if wrapped.Underlying != nil {
			errMsg += ": " + wrapped.Underlying.Error()
		}
	} else {
		errMsg = err.Error()
	}

	return &Error{
		Code:       code,
		Message:    message + ": " + errMsg,
		Underlying: err,
	}
}

func Is(err error, target error) bool {
	if err == nil || target == nil {
		return false
	}

	if e, ok := err.(*Error); ok {
		if t, ok := target.(*Error); ok {
			return e.Code == t.Code
		}
	}

	return err.Error() == target.Error()
}

func IsExitCode(err error, code ExitCode) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(*Error); ok {
		return e.Code == code
	}

	return false
}

// Handle processes an error, prints it to stderr, and exits the program.
// Deprecated: Use HandleReturn instead for library code. Handle is kept for
// backward compatibility but will be removed in a future version.
func Handle(err error) {
	if err == nil {
		return
	}

	var exitCode ExitCode = ExitCodeGeneral
	var message string
	var suggestion string

	if e, ok := err.(*Error); ok {
		exitCode = e.Code
		message = e.Message
		suggestion = e.Suggestion

		if e.Underlying != nil {
			logger.Error().Err(e.Underlying).Msg(e.Message)
		} else {
			logger.Error().Msg(e.Message)
		}
	} else {
		message = err.Error()
		logger.Error().Msg(message)
	}

	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	fmt.Fprintln(os.Stderr)
	red.Fprint(os.Stderr, "Error: ")
	fmt.Fprintln(os.Stderr, message)

	if suggestion != "" {
		yellow.Fprint(os.Stderr, "Suggestion: ")
		// Handle multi-line suggestions
		lines := strings.Split(suggestion, "\n")
		for i, line := range lines {
			if i == 0 {
				fmt.Fprintln(os.Stderr, line)
			} else {
				// Indent continuation lines
				if strings.HasPrefix(line, "  -") {
					cyan.Fprintln(os.Stderr, line)
				} else {
					fmt.Fprintln(os.Stderr, "           "+line)
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr)

	os.Exit(int(exitCode))
}

// HandleReturn processes an error and returns the appropriate exit code.
// Unlike Handle, it does not call os.Exit - the caller is responsible for
// exiting the program. This makes it suitable for use in library code.
func HandleReturn(err error) ExitCode {
	if err == nil {
		return ExitCodeSuccess
	}

	var exitCode ExitCode = ExitCodeGeneral
	var message string
	var suggestion string

	if e, ok := err.(*Error); ok {
		exitCode = e.Code
		message = e.Message
		suggestion = e.Suggestion

		if e.Underlying != nil {
			logger.Error().Err(e.Underlying).Msg(e.Message)
		} else {
			logger.Error().Msg(e.Message)
		}
	} else {
		message = err.Error()
		logger.Error().Msg(message)
	}

	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	fmt.Fprintln(os.Stderr)
	red.Fprint(os.Stderr, "Error: ")
	fmt.Fprintln(os.Stderr, message)

	if suggestion != "" {
		yellow.Fprint(os.Stderr, "Suggestion: ")
		lines := strings.Split(suggestion, "\n")
		for i, line := range lines {
			if i == 0 {
				fmt.Fprintln(os.Stderr, line)
			} else {
				if strings.HasPrefix(line, "  -") {
					cyan.Fprintln(os.Stderr, line)
				} else {
					fmt.Fprintln(os.Stderr, "           "+line)
				}
			}
		}
	}

	fmt.Fprintln(os.Stderr)

	return exitCode
}

// HandleQuiet processes an error quietly (minimal output) and exits the program.
// Deprecated: Use HandleQuietReturn instead for library code. HandleQuiet is kept
// for backward compatibility but will be removed in a future version.
func HandleQuiet(err error) {
	if err == nil {
		return
	}

	var exitCode ExitCode = ExitCodeGeneral

	if e, ok := err.(*Error); ok {
		exitCode = e.Code
	} else {
		logger.Error().Err(err).Msg("operation failed")
	}

	os.Exit(int(exitCode))
}

// HandleQuietReturn processes an error quietly and returns the appropriate exit code.
// Unlike HandleQuiet, it does not call os.Exit - the caller is responsible for
// exiting the program. This makes it suitable for use in library code.
func HandleQuietReturn(err error) ExitCode {
	if err == nil {
		return ExitCodeSuccess
	}

	var exitCode ExitCode = ExitCodeGeneral

	if e, ok := err.(*Error); ok {
		exitCode = e.Code
	} else {
		logger.Error().Err(err).Msg("operation failed")
	}

	return exitCode
}

func UserError(code ExitCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func UserErrorWithSuggestion(code ExitCode, message string, suggestion string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
	}
}

func APIError(err error) *Error {
	return &Error{
		Code:       ExitCodeAPIRequest,
		Message:    "API request failed",
		Underlying: err,
	}
}

func AuthError() *Error {
	return &Error{
		Code:       ExitCodeAPIAuth,
		Message:    "Authentication failed. Check your Azure DevOps personal access token.",
		Suggestion: "Set AZURE_PAT environment variable or add it to your config file (~/.config/adoctl/config.yaml)",
	}
}

func NotFoundError(resource string) *Error {
	return &Error{
		Code:       ExitCodeAPINotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Suggestion: "Verify the resource exists and you have access to it.",
	}
}

func NotFoundErrorWithSuggestions(resource string, suggestions []string) *Error {
	suggestionText := "Verify the resource exists and you have access to it."
	if len(suggestions) > 0 {
		suggestionText += "\n\nDid you mean:\n"
		for _, s := range suggestions {
			suggestionText += fmt.Sprintf("  - %s\n", s)
		}
	}
	return &Error{
		Code:       ExitCodeAPINotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Suggestion: suggestionText,
	}
}

func RepositoryNotFoundError(repoName string, similarRepos []string) *Error {
	suggestionText := "Use 'adoctl repos' to list available repositories."
	if len(similarRepos) > 0 {
		suggestionText = "Did you mean:\n"
		for _, r := range similarRepos {
			suggestionText += fmt.Sprintf("  - %s\n", r)
		}
		suggestionText += "\nOr use 'adoctl repos' to see all repositories."
	}
	return &Error{
		Code:       ExitCodeAPINotFound,
		Message:    fmt.Sprintf("Repository '%s' not found", repoName),
		Suggestion: suggestionText,
	}
}

func PRNotFoundError(prID int) *Error {
	return &Error{
		Code:       ExitCodeAPINotFound,
		Message:    fmt.Sprintf("Pull request #%d not found", prID),
		Suggestion: "Check the PR number and verify it exists in the repository.",
	}
}

func WorkItemNotFoundError(id int) *Error {
	return &Error{
		Code:       ExitCodeAPINotFound,
		Message:    fmt.Sprintf("Work item %d not found", id),
		Suggestion: "Verify the work item ID is correct and you have access to it.",
	}
}

func ConfigError(message string) *Error {
	return &Error{
		Code:       ExitCodeConfig,
		Message:    message,
		Suggestion: "Check your configuration file or set the required environment variables.",
	}
}

func ValidationError(message string) *Error {
	return &Error{
		Code:    ExitCodeValidation,
		Message: message,
	}
}

func TimeoutError(operation string) *Error {
	return &Error{
		Code:       ExitCodeTimeout,
		Message:    fmt.Sprintf("Operation timed out: %s", operation),
		Suggestion: "Try again with a longer timeout using --timeout flag.",
	}
}

func CancelledError(operation string) *Error {
	return &Error{
		Code:       ExitCodeCancellation,
		Message:    fmt.Sprintf("Operation cancelled: %s", operation),
		Suggestion: "The operation was interrupted. No changes were made.",
	}
}

// CommandError wraps errors from command handlers with consistent formatting.
// It preserves the original error chain for inspection while providing
// a user-friendly message.
func CommandError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// WrapWithMessage wraps an error with a message and returns an Error with the specified exit code.
// If the error is nil, it returns nil. If the error is already an Error, it preserves the code.
func WrapWithMessage(code ExitCode, message string, err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return &Error{
			Code:       e.Code,
			Message:    message + ": " + e.Message,
			Underlying: e.Underlying,
			Suggestion: e.Suggestion,
		}
	}

	return &Error{
		Code:       code,
		Message:    message,
		Underlying: err,
	}
}
