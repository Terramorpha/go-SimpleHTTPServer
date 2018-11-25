package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var Testing = "false"

var MainServer *mainHandler

func main() {

	var (
		err error
	)
	if MainConfig.Get("IsVerbose").Bool() {
		iPrintf("verbosity level: %d\n", MainConfig.Get("Verbosity").Int()) //on dit le niveau de verbosité
		iPrintf("mode: %s\n", MainConfig.Get("Mode").String())
	}

	if MainConfig.Get("IsAuth").Bool() {
		iPrintf("auth enabled\nUsername:%s\nPassword:%s\n", MainConfig.Get("UserName").String(), MainConfig.Get("Password").String())
	}
	MainServer = new(mainHandler) //l'endroit où le serveur stockera ses variables tel que le nb de connections
	MainServer.logBuffer = NewF(make([]byte, 1<<16))
	a := NewF(make([]byte, 8192))
	use(a)
	server := &http.Server{
		Addr:              ":" + MainConfig.Get("Port").String(),
		Handler:           MainServer,
		ReadHeaderTimeout: MainConfig.Get("RequestTimeout").Duration(),
		ReadTimeout:       MainConfig.Get("RequestTimeout").Duration(),
		WriteTimeout:      MainConfig.Get("RequestTimeout").Duration(),
		IdleTimeout:       MainConfig.Get("RequestTimeout").Duration(),
	}
	server.SetKeepAlivesEnabled(MainConfig.Get("IsKeepAlive").Bool())
	addrs := GetAddress()
	iPrintln("you can connect to this server on:")
	for _, v := range addrs {
		fmt.Printf("        "+"http://%s/\n", net.JoinHostPort(v.String(), MainConfig.Get("Port").String()))
	}
	vPrintf(1, "Server Config: %v\n", MainConfig)
	done := ManageServer(server) //manageserver permet de faire runner le server pi de savoir quand il est fermé
	//server.RegisterOnShutdown(func() {  }) //quoi faire quand le serveur ferme
	iPrintf("serving %s on port %s\n", MainConfig.Get("WorkingDir").String(), MainConfig.Get("Port").String())

	if MainConfig.Get("IsUI").Bool() {
		go WebUI()

	}
	if MainConfig.Get("IsTLS").Bool() {
		err = server.ListenAndServeTLS(MainConfig.Get("PathCertFile").String(), MainConfig.Get("PathKeyFile").String())
	} else {
		err = server.ListenAndServe()
	}
	if err != http.ErrServerClosed { //sert les requêtes http sur le listener et le stockage choisi
		Fatal(err)
	}
	<-done
}

func init() { //preparing server and checking
	GetFlags()
	CheckFlags()

}

type mainHandler struct {
	//ReqCount is a simple tracker for the number of http Request mainHandler has received
	Requests  int
	Succeeded int
	logBuffer io.ReadWriteSeeker
}

//Log
//implements a basic logging system. not fully useful yet.
func (m *mainHandler) Log(x ...interface{}) {
	fmt.Fprintf(m.logBuffer, "[%s] %s\n", time.Now().UTC().Format(http.TimeFormat), fmt.Sprint(x...))
}

//NewRequest increments the request counter. used for debugging and parsing logs.
//
//maybe.
func (m *mainHandler) NewRequest() int { //assigns a request number
	a := m.Requests
	m.Requests++
	return a
}

type FormFile struct {
	Data           []byte
	FileName       string
	WantedFileName string
}

func RenderHeader(h *http.Header) string {
	o := ""
	for i, v := range *h {
		o += fmt.Sprintf("%s: %v\n", i, v)
	}
	return o

}

func SendStatusFail(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	fmt.Fprintln(w, code, http.StatusText(code))
}

func ManageCli() chan int {
	c := make(chan int)
	go func() {
		b := bufio.NewReader(os.Stdin)
		for {
			s, err := b.ReadString('\n')
			if err != nil {
				panic(err)
			}
			args := ParseCommandLine(s)
			fmt.Println(args)
		}
	}()

	return c
}

/*
comment la page de settings va fonctionner:


(frontend)

	à gauche            à droite
	tt les settings     les erreurs de values






	[CONFIRM]



(backend)

	client            serveur
	------changements------->
	<-----erreurs/ok---------

	pour savoir si une valeur est valide, on utilise des types:
	types:
	- ipport
		(ip/hostname):(port number)
	- dir (selected by ui)

	- bool
		(true/false)
	- duration
		(chiffre)(m/s/h/µs/ms)

*/

func CheckSettingValid(x *Option) (*Option, error) {
	switch x.Type {
	case SettingTypeBool:
		if StringInArray(x.Value, StringValueBool) {
			return x, nil
		}
		return nil, errors.New("invalid boolean value. Valid booleans are:\n" + strings.Join(StringValueBool, " ,"))
	case SettingTypeDuration:
		_, err := time.ParseDuration(x.Value)
		if err != nil {
			return nil, err
		}
		return x, nil
	case SettingTypeFile:
		//TODO
	case SettingTypeBindAddr:
		_, _, err := net.SplitHostPort(x.Value)
		if err != nil {
			return nil, err
		}
		return x, err
	case SettingTypePort:
		_, err := net.LookupPort("tcp", x.Value)
		if err != nil {
			return nil, err
		}
		return x, nil
	default:
		return nil, nil
	}
	return nil, nil
}
