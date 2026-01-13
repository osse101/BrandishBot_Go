package handler

import (
	"net/http"
	"strconv"
)

func getQueryInt(r *http.Request, key string, defaultValue int) int {
	if valStr := r.URL.Query().Get(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			return val
		}
	}
	return defaultValue
}
