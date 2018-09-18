package main

import (
	"fmt"
	"net/http"
)

var (
	wd = ""
)

func main() {
	http.HandleFunc("/", serve)
	http.ListenAndServe(":8080", nil)
}

func serve(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	http.ServeFile(w, r, "/home/terramorpha"+r.URL.Path)
}
