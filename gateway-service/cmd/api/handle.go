package main

import (
	"net/http"
)

func (app *Config) usage(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will Write Usage Here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) googleLogin(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will Write Usage Here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}
