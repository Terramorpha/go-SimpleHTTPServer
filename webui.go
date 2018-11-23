package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
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
	if r.URL.Path == "/settings.json" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(MainConfig)
		return
	}
	page := BasicHTMLFile("", "marmelade", "")
	w.Write([]byte(page))
}

func JsSettingsRenderer() []byte {

}

var jsr = []byte(`


`)
