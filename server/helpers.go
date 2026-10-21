package server

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
)

type envelope map[string]any

func (app *Server) background(fn func()) {
	app.wg.Go(func() {
		defer func() {
			pv := recover()
			if pv != nil {
				app.logger.Error(fmt.Sprintf("%v", pv))
			}
		}()
		fn()
	})
}

func (app *Server) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	opts := []json.Options{
		json.Deterministic(true),
		jsontext.Multiline(true),
	}
	js, err := json.Marshal(data, opts...)
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *Server) sentResponse(w http.ResponseWriter, r *http.Request, sender, target string) {
	if target != "" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"email_status": "sent", "email_sender": sender}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
