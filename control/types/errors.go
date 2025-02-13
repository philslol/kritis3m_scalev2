package types

import (
	"errors"
	"net/http"
)

var (
	// ErrInvalidParam either means the given route parameter was wrong, like a non uint, or too long
	ErrInvalidParam  = &RequestError{ErrorString: "bad request", ErrorCode: http.StatusBadRequest}
	ErrInternalError = &RequestError{ErrorString: "internal error", ErrorCode: http.StatusInternalServerError}
	ErrNotFound      = &RequestError{ErrorString: "request not found", ErrorCode: http.StatusNotFound}
	// ErrUnauthorized means the user could not be validated and any JWT tokens on client side should be removed
	ErrUnauthorized = &RequestError{ErrorString: "unauthorized", ErrorCode: http.StatusUnauthorized}
	// ErrForbidden is either anon accessing a route that requires auth, or an authed user without the correct permissions
	ErrForbidden = &RequestError{ErrorString: "forbidden", ErrorCode: http.StatusForbidden}

	ErrNoSha = errors.New("no Sha provided, can't find matching node")
	ErrNoID  = errors.New("no ID provided, can't find matching node")

	ErrWrongParameter = errors.New("provided parameter is in wrong format check type and sign")

	ErrNodeNotFound = errors.New("no node matching the provided identifier")
)

// RequestError holds the message string and http code
type RequestError struct {
	ErrorString string
	ErrorCode   int
}

// Code returns the http error code
func (err *RequestError) Code() int {
	return err.ErrorCode
}

func (err *RequestError) Error() string {
	return err.ErrorString
}

// ErrorMessage returns the code and message for Gins JSON helpers
func ErrorMessage(errorType *RequestError) (code int, message map[string]interface{}) {
	return errorType.Code(), map[string]interface{}{"error_message": errorType.Error()}
}
