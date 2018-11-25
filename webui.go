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
	"strings"
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
		FrontendGetJs(w)
		return
	case "/settings.json":
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.Encode(MainConfig)
		return
	case "/":
		FrontendGetHtml(w)
		return
	default:
	}
}

func JsSettingsRenderer() []byte {
	return nil
}

func (s *WebUISettings) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	a, _ := ioutil.ReadAll(r.Body)
	//fmt.Println(r.Header)
	defer r.Body.Close()
	fmt.Fprintln(os.Stdout, string(a))
	split := strings.Split(string(a), "=")
	if len(split) != 2 {
		return
	}
	k, v := split[0], split[1]
	oldv := MainConfig.Get(k).String()
	MainConfig.Set(k, v)
	if oldv != v {
		iPrintf("%s changed from \"%s\" to \"%s\"\n", k, Colorize(oldv, ColorMagenta), Colorize(v, ColorMagenta))
	}
}

var jsr = []byte("")

func FrontendGetJs(w http.ResponseWriter) {
	switch Testing {
	case "true":
		f, err := os.Open("frontend.js")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/javascript")
		io.Copy(w, f)
	case "false":
		io.Copy(w, bytes.NewReader([]byte(JsFrontend)))
	}
}

func FrontendGetHtml(w http.ResponseWriter) {
	switch Testing {
	case "true":
		f, err := os.Open("frontend.js")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/html")
		io.Copy(w, f)
	case "false":
		io.Copy(w, bytes.NewReader([]byte(HtmlFrontend)))
	}
}
