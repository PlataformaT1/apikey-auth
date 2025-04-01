package errorx

import (
	"fmt"
)

type (
	// ErrorCode defines supported error codes.
	ErrorCode uint

	// Error interface represents an wrap error.
	Error interface {
		Error() string
		SetOrigin(orig error) Error
		Unwrap() error
		Code() ErrorCode
		Message() string
		SetMessage(message string) Error
	}

	// ObjectError is the default wrap error
	// that implements the Error interface.
	ObjectError struct {
		orig error
		msg  string
		code ErrorCode
	}
)

// WrapErrorf returns a wrapped error.
func WrapErrorf(orig error, code ErrorCode, format string, a ...interface{}) Error {
	return &ObjectError{
		orig: orig,
		code: code,
		msg:  fmt.Sprintf(format, a...),
	}
}

// NewErrorf instantiates a new error.
func NewErrorf(code ErrorCode, format string, a ...interface{}) Error {
	return WrapErrorf(nil, code, format, a...)
}

// Error returns the message, when wrapping errors the wrapped error is returned.
func (e *ObjectError) Error() string {
	if e.orig != nil {
		return fmt.Sprintf("%s : %v", e.msg, e.orig)
	}

	return e.msg
}

// SetError set the error's origin.
func (e *ObjectError) SetOrigin(orig error) Error {
	e.orig = orig
	return e
}

// Unwrap returns the wrapped error, if any.
func (e *ObjectError) Unwrap() error {
	return e.orig
}

// Code returns the code representing this error.
func (e *ObjectError) Code() ErrorCode {
	return e.code
}

// Message return the error's message.
func (e *ObjectError) Message() string {
	return e.msg
}

// SetMessage set the error's message.
func (e *ObjectError) SetMessage(message string) Error {
	e.msg = message
	return e
}
