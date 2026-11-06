package problem

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/a-pelikh/tripgo/api"
)

func Write(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}
