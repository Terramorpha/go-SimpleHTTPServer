package main

import (
	"encoding/json"
	"net/http"
)

func (s *WebUISettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/settings.json" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(s.Settings)
	}
	page := BasicHTMLFile("", "", "")

	use(page)
}
