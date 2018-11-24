package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func WebUI() {
	var (
		err error
	)
	var (
		listeningPort string = "8081"
		logContent           = make([]byte, 8192)
		logWriter            = bytes.NewBuffer(logContent)
		logger               = log.New(logWriter, "", 0)
	)

	var handler *WebUISettings
	serv := http.Server{
		Addr:              "localhost:" + listeningPort,
		Handler:           handler,
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		ErrorLog:          logger,
	}
	iPrintf("starting web ui on :%s\n", MainConfig.Get("UiPort").String())
	go func() {
		if MainConfig.Get("IsTLS").Bool() {
			err = serv.ListenAndServeTLS(MainConfig.Get("PathCertFile").String(), MainConfig.Get("PathKeyFile").String())
		} else {
			err = serv.ListenAndServe()
		}
		Fatal(err)
		//if err != http.ErrServerClosed {
		//}

	}()
}

type WebUISettings struct {
	//placeholder
}

func (s *WebUISettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.UpdateConfig(w, r)
	}
	switch r.URL.Path {
	case "/frontend.js":
		f, err := os.Open("frontend.js")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/javascript")
		io.Copy(w, f)
		return
	case "/settings.json":
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.Encode(MainConfig)
		return
	case "/":
		f, err := os.Open("frontend.html")
		if err != nil {
			wPrintln(err)
			return
		}
		defer f.Close()
		io.Copy(w, f)
		//page := BasicHTMLFile("", "marmelade", "/frontend.js")
		w.Header().Set("Content-Type", "text/html")

		return
	default:
	}
}

func JsSettingsRenderer() []byte {
	return nil
}

func (s *WebUISettings) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got post")
	a, _ := ioutil.ReadAll(r.Body)
	//fmt.Println(r.Header)
	fmt.Fprintln(os.Stdout, string(a))
	r.Body.Close()
}

var jsr = []byte("")
