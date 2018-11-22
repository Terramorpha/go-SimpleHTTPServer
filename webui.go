package main

import (
	"encoding/json"
	"net/http"
)

func Get(set *[]*Setting, x string) *Setting {
	for i := range *set {
		if (*set)[i].Name == x {
			return (*set)[i]
		}
	}
	return nil
}

func Set(set *[]*Setting, x string, value string) *Setting {
	for i := range *set {
		if (*set)[i].Name == x {
			(*set)[i].Value = value
		}
	}
	return nil
}

func (s *WebUISettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/settings.json" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(s.Settings)
	}
	page := BasicHTMLFile("", "", "")

	use(page)
}
