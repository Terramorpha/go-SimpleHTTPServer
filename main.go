package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var ( //where we put flag variables and global variables

	port    string
	portNum int
	//WorkingDir is the root of the server
	WorkingDir string
	//PathCertFile is the file from which http.ListenAndServeTLS will get its certificates
	PathCertFile string
	//PathKeyFile is the file from which http.ListenAndServeTLS will get its encryption keys
	PathKeyFile string

	verbosityLevel  int
	mode            string
	shutdowmTimeout time.Duration
	requestTimeout  time.Duration

	authPassword string
	authUsername string
	webUIport    int

	isVerbose   bool
	isTLS       bool
	isKeepAlive bool
	isDebug     bool
	isAuth      bool
	isTellTime  bool
	isWebUI     bool
	isGitCommit bool
	isColored   bool
)

var webUI = []*Setting{
	&Setting{
		Name:  "WorkingDir",
		Type:  "file",
		Value: ".",
	},
	&Setting{
		Name:  "PathCertFile",
		Type:  "file",
		Value: "",
	},
	&Setting{
		Name:  "PathKeyFile",
		Type:  "file",
		Value: "",
	},
	&Setting{
		Name:  "Mode",
		Type:  "string",
		Value: "",
	},
	&Setting{
		Name:  "ShutdownTimeout",
		Type:  "duration",
		Value: "10s",
	},
	&Setting{
		Name:  "RequestTimeout",
		Type:  "duration",
		Value: "10s",
	},
	&Setting{
		Name:  "AuthPassword",
		Type:  "string",
		Value: "",
	},
	&Setting{
		Name:  "AuthUsername",
		Type:  "string",
		Value: "",
	},
	&Setting{
		Name:  "IsTLS",
		Type:  "bool",
		Value: "false",
	},
	&Setting{
		Name:  "IsKeepAlive",
		Type:  "bool",
		Value: "true",
	},
	&Setting{
		Name:  "IsAuth",
		Type:  "bool",
		Value: "false",
	},
}

func main() {
	if isVerbose {
		iPrintf("verbosity level: %d\n", verbosityLevel) //on dit le niveau de verbosité
		iPrintf("mode: %s\n", mode)
	}

	if isAuth {
		iPrintf("auth enabled\nUsername:%s\nPassword:%s\n", authUsername, authPassword)
	}
	han := new(mainHandler) //l'endroit où le serveur stockera ses variables tel que le nb de connections
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           han,
		ReadHeaderTimeout: requestTimeout,
		ReadTimeout:       requestTimeout,
		WriteTimeout:      requestTimeout,
		IdleTimeout:       requestTimeout,
	}
	server.SetKeepAlivesEnabled(isKeepAlive)
	addrs := GetAddress()
	iPrintln("you can connect to this server on:")
	for _, v := range addrs {
		fmt.Printf("        "+"http://%s/\n", net.JoinHostPort(v.String(), strconv.Itoa(portNum)))
	}
	done := ManageServer(server) //manageserver permet de faire runner le server pi de savoir quand il est fermé
	//server.RegisterOnShutdown(func() {  }) //quoi faire quand le serveur ferme
	iPrintf("serving %s on port %s\n", WorkingDir, port)
	WebUI()
	var (
		err error
	)
	if isTLS {
		err = server.ListenAndServeTLS(PathCertFile, PathKeyFile)
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
	logBuffer string
}

//Log
//implements a basic logging system. not fully useful yet.
func (m *mainHandler) Log(x ...interface{}) {
	m.logBuffer += fmt.Sprintf("[%s] %s\n", time.Now().UTC().Format(http.TimeFormat), fmt.Sprint(x...))

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

	handler := new(WebUISettings)
	handler.Settings = webUI
	serv := http.Server{
		Addr:              "localhost:" + listeningPort,
		Handler:           handler,
		TLSConfig:         nil,
		ReadTimeout:       requestTimeout,
		ReadHeaderTimeout: requestTimeout,
		WriteTimeout:      requestTimeout,
		IdleTimeout:       requestTimeout,
		ErrorLog:          logger,
	}
	go func() {
		err = serv.ListenAndServe()
		Fatal(err)
		//if err != http.ErrServerClosed {
		//}

	}()
}

type WebUISettings struct {

	//WorkingDir is the root of the server
	WorkingDir string
	//PathCertFile is the file from which http.ListenAndServeTLS will get its certificates
	PathCertFile string
	//PathKeyFile is the file from which http.ListenAndServeTLS will get its encryption keys
	PathKeyFile        string
	mode               string
	shutdowmTimeout    time.Duration
	requestTimeout     time.Duration
	authPassword       string
	authUsername       string
	isTLS              bool
	isKeepAliveEnabled bool
	isAuthEnabled      bool

	Settings []*Setting
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

type Setting struct {
	Name  string
	Type  string
	Value string
}

func CheckSettingValid(x *Setting) (*Setting, error) {
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
	case SettingTypeIpPort:
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

const (
	SettingTypeIpPort   = "ipport"
	SettingTypePort     = "port"
	SettingTypeFile     = "file"
	SettingTypeBool     = "bool"
	SettingTypeDuration = "duration"
	SettingTypeString   = "string"
)

const SettingsJS = ``
