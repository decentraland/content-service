package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Error interface {
	error
	Status() int
}

type StatusError struct {
	Code int
	Err  error
}

type ErrorMsg struct {
	Error string `json:"error"`
}

func (se StatusError) Error() string {
	return se.Err.Error()
}

func (se StatusError) Status() int {
	return se.Code
}

type Response interface {
	WriteResponse(w http.ResponseWriter) error
}

type JsonResponse struct {
	StatusCode int
	Content    interface{}
	Headers    map[string]string
}

type Handler struct {
	Ctx interface{}
	H   func(ctx interface{}, w http.ResponseWriter, r *http.Request) error
}

type ResponseHandler struct {
	Ctx interface{}
	H   func(ctx interface{}, r *http.Request) (Response, error)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.H(h.Ctx, w, r)
	if err != nil {
		handleError(w, err)
	}
}

func NewOkJsonResponse(content interface{}) *JsonResponse {
	return &JsonResponse{StatusCode: http.StatusOK, Content: content, Headers: nil}
}

func NewOkEmptyResponse() *JsonResponse {
	return &JsonResponse{StatusCode: http.StatusOK, Content: map[string]string{}, Headers: nil}
}

func NewBadRequestError(msg string) *StatusError {
	return WrapInBadRequestError(errors.New(msg))
}

func NewInternalError(msg string) *StatusError {
	return WrapInInternalError(errors.New(msg))
}

func NewNotFoundError(msg string) *StatusError {
	return &StatusError{http.StatusNotFound, errors.New(msg)}
}

func WrapInBadRequestError(err error) *StatusError {
	return &StatusError{http.StatusBadRequest, err}
}

func WrapInInternalError(err error) *StatusError {
	return &StatusError{http.StatusInternalServerError, err}
}

func (h ResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, err := h.H(h.Ctx, r)
	if err != nil {
		handleError(w, err)
	} else {
		err = response.WriteResponse(w)
		if err != nil {
			unexpectedError(w, err)
		}
	}
}

func handleError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case Error:
		writeJsonError(w, e.Status(), e.Error())
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

func writeJsonError(w http.ResponseWriter, code int, msg string) {
	errorJSON, err := json.Marshal(ErrorMsg{Error: msg})
	if err != nil {
		unexpectedError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(errorJSON)
	if err != nil {
		unexpectedError(w, err)
	}
}

func unexpectedError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(500), 500)
}

func (r *JsonResponse) WriteResponse(w http.ResponseWriter) error {
	contentsJSON, err := json.Marshal(r.Content)
	if err != nil {
		return WrapInInternalError(err)
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Headers != nil {
		for k, v := range r.Headers {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(r.StatusCode)
	_, err = w.Write(contentsJSON)
	if err != nil {
		return WrapInInternalError(err)
	}
	return nil
}
