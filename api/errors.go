package api

import (
	"fmt"
	"net/http"
)

func InternalServerError(w http.ResponseWriter, err error) {
	writeErrorRequest(w, http.StatusInternalServerError, err)
}

func BadRequestError(w http.ResponseWriter, err error) {
	writeErrorRequest(w, http.StatusBadRequest, err)
}

func writeErrorRequest(w http.ResponseWriter, code int, err error) {
	w.WriteHeader(code)
	fmt.Fprintf(w, "%s: %s", http.StatusText(code), err.Error())
}
