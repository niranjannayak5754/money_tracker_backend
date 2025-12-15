package apperr

import "fmt"

type Kind string

const (
	Unauthorized Kind = "unauthorized"
	Forbidden    Kind = "forbidden"
	NotFound     Kind = "not_found"
	Conflict     Kind = "conflict"
	Validation   Kind = "validation"
	Internal     Kind = "internal"
)

type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Validation error (bad input, business rule violation)
func ValidationErr(msg string) error {
	return &Error{
		Kind:    Validation,
		Message: msg,
	}
}

// Unauthorized error (auth missing / invalid)
func UnauthorizedErr(msg string) error {
	return &Error{
		Kind:    Unauthorized,
		Message: msg,
	}
}

// Forbidden error (auth ok but action not allowed)
func ForbiddenErr(msg string) error {
	return &Error{
		Kind:    Forbidden,
		Message: msg,
	}
}

// Not found error
func NotFoundErr(msg string) error {
	return &Error{
		Kind:    NotFound,
		Message: msg,
	}
}

// Conflict error (duplicate, already exists, etc.)
func ConflictErr(msg string) error {
	return &Error{
		Kind:    Conflict,
		Message: msg,
	}
}

// Internal error (wrap real cause)
func InternalErr(msg string, cause error) error {
	return &Error{
		Kind:    Internal,
		Message: msg,
		Cause:   cause,
	}
}

// IsKind checks if an error is a specific app error kind
func IsKind(err error, kind Kind) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*Error); ok {
		return ae.Kind == kind
	}
	return false
}

// AsAppError safely extracts *Error
func AsAppError(err error) (*Error, bool) {
	ae, ok := err.(*Error)
	return ae, ok
}
