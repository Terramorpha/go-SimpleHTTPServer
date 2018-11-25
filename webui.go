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
	"strconv"
	"strings"
	"time"
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
	SelectedOption := MainConfig.Get(k)
	oldValue := SelectedOption.String()
	switch SelectedOption.Type { //switch to verify validity of update
	default:
		fallthrough
	case OptionTypeString:
		SelectedOption.SetString(v)
	case OptionTypeBool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			vPrintf(1, "couldn't update value %s to %s, %v\n", k, v, err)
			return
		}
		SelectedOption.SetBool(b)
	case OptionTypeDuration:
		d, err := time.ParseDuration(v)
		if err != nil {
			vPrintf(1, "couldn't update value %s to %s, %v\n", k, v, err)
			return
		}
		SelectedOption.SetString(d.String())
	}
	if oldValue != v {
		iPrintf("%s changed from \"%s\" to \"%s\"\n", k, Colorize(oldValue, ColorMagenta), Colorize(v, ColorMagenta))
	}
}

var jsr = []byte("")

const webuiFolder = "webui"

func FrontendGetJs(w http.ResponseWriter) {
	switch Testing {
	case "true":
		f, err := os.Open(webuiFolder + "/" + "frontend.js")
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
		f, err := os.Open(webuiFolder + "/" + "frontend.html")
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
