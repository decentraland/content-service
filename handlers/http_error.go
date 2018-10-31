package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorMsg struct {
	Error string `json:"error"`
}

func handle400(w http.ResponseWriter, code int, msg string) {
	errorJSON, err := json.Marshal(ErrorMsg{Error: msg})
	if err != nil {
		handle500(w, err)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(errorJSON)
	if err != nil {
		handle500(w, err)
	}
}

func handle500(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(500), 500)
}
