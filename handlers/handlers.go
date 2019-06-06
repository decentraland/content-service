package handlers

import (
	"encoding/json"
	"errors"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"
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
	Ctx   interface{}
	H     func(ctx interface{}, w http.ResponseWriter, r *http.Request) error
	Id    string
	Agent *metrics.Agent
}

type ResponseHandler struct {
	Ctx   interface{}
	H     func(ctx interface{}, r *http.Request) (Response, error)
	Id    string
	Agent *metrics.Agent
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Request received at endpoint: %s", h.Id)
	tx := h.Agent.EndpointMetrics(h.Id, w, r)
	defer tx.Close()

	err := h.H(h.Ctx, w, r)
	if err != nil {
		tx.ReportError(err)
		log.Error(err)
		handleError(w, err)
	}
}

func (h ResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Request received at endpoint: %s", h.Id)
	tx := h.Agent.EndpointMetrics(h.Id, w, r)
	defer tx.Close()

	response, err := h.H(h.Ctx, r)
	if err != nil {
		tx.ReportError(err)
		log.Error(err)
		handleError(w, err)
	} else {
		err = response.WriteResponse(w)
		if err != nil {
			unexpectedError(w, err)
		}
	}
}

func NewOkJsonResponse(content interface{}) *JsonResponse {
	return &JsonResponse{StatusCode: http.StatusOK, Content: content, Headers: nil}
}

func NewOkEmptyResponse() Response {
	return &JsonResponse{StatusCode: http.StatusOK, Content: nil, Headers: nil}
}

func NewBadRequestError(msg string) error {
	return WrapInBadRequestError(errors.New(msg))
}

func NewInternalError(msg string) error {
	return WrapInInternalError(errors.New(msg))
}

func NewNotFoundError(msg string) error {
	return StatusError{http.StatusNotFound, errors.New(msg)}
}

func WrapInBadRequestError(err error) error {
	return StatusError{http.StatusBadRequest, err}
}

func WrapInInternalError(err error) error {
	return StatusError{http.StatusInternalServerError, err}
}

func ExtractContentFormJsonRequest(r *http.Request, c interface{}, v validation.Validator) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(c); err != nil {
		return WrapInBadRequestError(err)
	}
	if err := v.ValidateStruct(c); err != nil {
		return WrapInBadRequestError(err)
	}
	return nil
}

func handleError(w http.ResponseWriter, err error) {
	log.Errorf("Error: %s", err.Error())
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
	log.Error(err)
	http.Error(w, http.StatusText(500), 500)
}

func (r *JsonResponse) WriteResponse(w http.ResponseWriter) error {
	var err error
	contentsJSON := []byte{}
	if r.Content != nil {
		contentsJSON, err = json.Marshal(r.Content)
		if err != nil {
			return WrapInInternalError(err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Headers != nil {
		for k, v := range r.Headers {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(r.StatusCode)
	err = nil
	if len(contentsJSON) > 0 {
		_, err = w.Write(contentsJSON)
	}
	if err != nil {
		return WrapInInternalError(err)
	}
	return nil
}
