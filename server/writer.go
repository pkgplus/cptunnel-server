package server

import (
	"encoding/json"
	"net/http"
)

func errorWriter(rw http.ResponseWriter, req *http.Request, code int, err error) {
	rw.WriteHeader(code)

	bytes, _ := json.Marshal(map[string]string{
		"message": err.Error(),
	})

	rw.Write(bytes)
	rw.Header().Set("Content-Type", "application/json")
}

func jsonWriter(rw http.ResponseWriter, req *http.Request, v interface{}) {
	bytes, _ := json.Marshal(v)
	rw.Write(bytes)
	rw.Header().Set("Content-Type", "application/json")
}
