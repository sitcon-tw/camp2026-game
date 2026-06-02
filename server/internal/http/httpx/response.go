package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type Error struct {
	Status  int
	Detail  string
	Details []ErrorDetail
}

func (e *Error) Error() string {
	return e.Detail
}

type ProblemDetails struct {
	Type     string        `json:"type,omitempty"`
	Title    string        `json:"title,omitempty"`
	Status   int           `json:"status,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	Instance string        `json:"instance,omitempty"`
	Errors   []ErrorDetail `json:"errors,omitempty"`
}

type ErrorDetail struct {
	Location string `json:"location,omitempty"`
	Message  string `json:"message,omitempty"`
}

func NewError(status int, detail string) error {
	return &Error{Status: status, Detail: detail}
}

func BadRequest(detail string) error {
	return NewError(http.StatusBadRequest, detail)
}

func NotFound(detail string) error {
	return NewError(http.StatusNotFound, detail)
}

func UnprocessableEntity(detail string, details ...ErrorDetail) error {
	return &Error{Status: http.StatusUnprocessableEntity, Detail: detail, Details: details}
}

func ServiceUnavailable(detail string) error {
	return NewError(http.StatusServiceUnavailable, detail)
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	detail := http.StatusText(status)

	var httpErr *Error
	if errors.As(err, &httpErr) {
		status = httpErr.Status
		detail = httpErr.Detail
	}

	if requestID := chimw.GetReqID(r.Context()); requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	body := ProblemDetails{
		Status: status,
		Title:  http.StatusText(status),
		Detail: detail,
	}
	if httpErr != nil && len(httpErr.Details) > 0 {
		body.Errors = httpErr.Details
	}

	_ = json.NewEncoder(w).Encode(body)
}
