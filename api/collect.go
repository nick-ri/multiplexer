package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/NickRI/multiplexer/collector"
)

func Collect(collector collector.Collector, urls, clim int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var request []string

		ctx := r.Context()

		defer r.Body.Close()

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			InternalServerError(w, err)
			return
		}

		if len(request) > urls {
			BadRequestError(w, errors.New("url list size is too big"))
			return
		}

		data, err := collector.Collect(ctx, request, clim)
		if err != nil {
			InternalServerError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if err := json.NewEncoder(w).Encode(data); err != nil {
			InternalServerError(w, err)
			return
		}
	}
}
