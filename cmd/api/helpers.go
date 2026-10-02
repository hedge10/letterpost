package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
)

type envelope map[string]any

func (app *application) background(fn func()) {
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

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
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
