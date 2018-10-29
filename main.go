package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const ( //terminal colors
	//ColorReset resets all colors/properties in terminal escape sequences
	ColorReset = 0
	//ColorBold makes text bold
	ColorBold       = 1
	ColorDim        = 2
	ColorUnderlined = 4
	ColorBlink      = 5
	ColorReversed   = 7
	ColorHidden     = 8

	ColorBlack   = 0
	ColorRed     = 1
	ColorGreen   = 2
	ColorYellow  = 3
	ColorBlue    = 4
	ColorMagenta = 5
	ColorCyan    = 6
	ColorGrey    = 7
)

var (
	gitCommit = ""
)

const (
	defaultWorkingDir = "."
)

const (
	//MaxDuration is the maximum duration time.Duration can take
	MaxDuration time.Duration = (1 << 63) - 1
)

var ( //error constants
	//ErrorUnauthorized is the error reporting an authentification error
	ErrorUnauthorized = errors.New("CheckAuth: client didn't provide correct authorization")
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

	isVerbose          bool
	isTLS              bool
	isKeepAliveEnabled bool
	isDebug            bool
	isAuthEnabled      bool
	isTellTime         bool
	isWebUIEnabled     bool
	isGitCommit        bool
	isColorEnabled     bool
)

func main() {
	if isVerbose {
		iPrintf("verbosity level: %d\n", verbosityLevel) //on dit le niveau de verbosité
		iPrintf("mode: %s\n", mode)
	}

	if isAuthEnabled {
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
	server.SetKeepAlivesEnabled(isKeepAliveEnabled)
	addrs := GetAddress()
	iPrintln("you can connect to this server on:")
	for _, v := range addrs {
		fmt.Printf("        "+"http://%s/\n", net.JoinHostPort(v.String(), strconv.Itoa(portNum)))
	}
	done := ManageServer(server) //manageserver permet de faire runner le server pi de savoir quand il est fermé
	//server.RegisterOnShutdown(func() {  }) //quoi faire quand le serveur ferme
	iPrintf("serving %s on port %s\n", WorkingDir, port)

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
	Flags()

	{ //setting auth if pass or user is set
		if authUsername != "" || authPassword != "" {
			isAuthEnabled = true
		}
	}
	//dPrintln(User)

	{ //checking mode string
		if mode != "" {
			switch mode {
			case "web":
			case "fileserver":
			default:
				Fatal("invalid mode. only these are valid:\n" + fmt.Sprint("web", "fileserver"))
			}
		}
	}
	{ //checking TLS settings
		if isTLS {
			if PathCertFile == "" {
				Fatal("path of certificate file must be given for TLS to work (-certfile)")
			}
			if PathKeyFile == "" {
				Fatal("path of key file must be given for TLS to work (-keyfile)")
			}
			certfile, err := os.OpenFile(PathCertFile, os.O_RDONLY, 0400)
			if err != nil {
				Fatal(fmt.Sprintf("certfile %s cant be accessed, TLS can't work", PathCertFile))
			}
			certfile.Close()
			keyfile, err := os.OpenFile(PathKeyFile, os.O_RDONLY, 0400)
			if err != nil {
				Fatal(fmt.Sprintf("keyfile %s cant be accessed, TLS can't work", PathKeyFile))
			}
			keyfile.Close()
		}

	}
	{ //checking working directory exists (or else nothing will get done)
		file, err := os.Open(WorkingDir)
		defer file.Close()
		if err != nil {

			switch {
			case os.IsNotExist(err):
				Fatal(fmt.Sprintf("directory %s doesn't exist: %v", WorkingDir, err))
			case os.IsPermission(err):
				Fatal(fmt.Sprintf("you don't have permission to open %s directory: %v", WorkingDir, err))
			case os.IsTimeout(err):
				Fatal(fmt.Sprintf("getting directory %s timed out: %v", WorkingDir, err))
			}

		}
		if stat, err := file.Stat(); err != nil {
			Fatal(fmt.Sprintf("error getting file stats: %v", err))
		} else {
			if !stat.IsDir() {
				Fatal(fmt.Sprintf("%s is not a directory", WorkingDir))
			}
		}

	}
	{ //port permission  && validity checking

		portnum, err := net.LookupPort("tcp", port)
		if err != nil {

			Fatal("error invalid port number: " + err.Error())
		}
		portNum = portnum
		if portNum < 1024 {
			wPrintf("need root to bind on port %d\n", portNum)
		}

	}

	{ //setting correct verbosity levels

		if verbosityLevel > 0 { //au cas ou l'utilisateur a pensé à metre le niveau de verbosité mais pas d'activer la verbosité
			isVerbose = true //on l'active
		}
		if isVerbose && verbosityLevel == 0 { //si c'est verbose, le niveau devrait être plus élevé que 0
			verbosityLevel = 1
		}

	}
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

//ServeHTTP separes different types of request.
func (m *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	switch r.Method {
	case "GET":
		m.ManageGET(w, r, true)
	case "HEAD":
		m.ManageHEAD(w, r)
	case "POST":
		m.ManagePOST(w, r)
	case "CONNECT":
		m.ManageCONNECT(w, r)
	default:
		iPrintln("Got request:", r.Method)
		iPrintln("header:", r.Header)
	}

}

//ManageCONNECT is not implemented yet.
//
//it is supposed (when it will be implemented)
//to give the functionnality
//of an HTTP proxy
func (m *mainHandler) ManageCONNECT(w http.ResponseWriter, r *http.Request) { //currently shit
	return
	dPrintf("%#+v\n", r.URL)

	conn, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	w.WriteHeader(http.StatusOK)
	io.Copy(w, conn)
}

//NewRequest increments the request counter. used for debugging and parsing logs.
//
//maybe.
func (m *mainHandler) NewRequest() int { //assigns a request number
	a := m.Requests
	m.Requests++
	return a
}

//ManageGET manages get requests.
//
//yeah.
//
//that's it
func (m *mainHandler) ManageGET(w http.ResponseWriter, r *http.Request, writeBody bool) { //serves get requests

	{ //setting default header
		w.Header().Add("Accept-Ranges", "bytes")
	}

	var (
		MimeType string
		id       = m.NewRequest() // request id
	)

	if isAuthEnabled { //basic auth
		err := CheckAuth(w, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			vPrintf(1, "failed auth from %v: %s\n", r.RemoteAddr, err.Error())
			return
		}

	}

	{ //logging
		log := fmt.Sprintf("[%d] got %s request to %v from %v\n", id, r.Method, r.RequestURI, r.RemoteAddr)
		finishedServing := fmt.Sprintf("finished serving [%d] at %v\n", id, r.RemoteAddr)
		m.Log(log)
		vPrintf(1, log)
		defer vPrintf(1, finishedServing)
		defer m.Log(1, finishedServing)
	}
	// the actual URL
	if isDebug {
		w.Header().Add("Request-Id", strconv.Itoa(id))
	}

	vPrintf(1, "[%d] asking for %v\n", id, r.URL.EscapedPath())

	ComposedPath := WorkingDir + path.Clean(r.URL.Path)
	//vPrintf(0, "%s\n", ComposedPath)
	if strings.Contains(r.RequestURI, "../") { // ../ permits a request to access files outside the server's scope
		//w.Header().Add("Connection", "close")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	//.. Check done
	// no ..
	//check if requested content exists
	File, err := os.Open(ComposedPath)
	if err != nil { //the file simply doesn't exist or is inaccessible
		vPrintf(1, "[%d] request failed: %v\n", id, err)
		//too informative//http.Error(w, err.Error(), http.StatusNotFound)
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dPrintf("file opened: %s\n", ComposedPath)
	//fmt.Println(*File)
	defer File.Close()
	//file exists

	//checks if file is a directory
	fileInfo, err := File.Stat()
	if err != nil { //cannot stat the file(os error)
		vPrintf(1, "[%d] request failed: %v\n", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fileInfo.IsDir() { //the path pointed to by the URL exists AND is a folder
		vPrintf(1, "%s is a folder\n", ComposedPath)
		var (
			lastModified string = time.Now().UTC().Format(http.TimeFormat)
			content      []byte
		)
		switch mode { // checks for index.html
		default:
			content, err = render(WorkingDir, r.URL.Path)
			if err != nil {
				http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
				vPrintf(1, "[%d] error rendering directory: %v\n", id, err)
				return
			}
			content = []byte(fmt.Sprintf(BasicHTMLFile(), "", string(content)))
		case "web":
			content, err = ioutil.ReadFile(ComposedPath + "/index.html")
			if err != nil {
				SendStatusFail(w, http.StatusNotFound)
				//http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
				return
			}
		case "fileserver":
			renderedFolder, err := render(WorkingDir, r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
				return
			}
			content = []byte(fmt.Sprintf(BasicHTMLFile(), "", BasicFileServerHeader()+string(renderedFolder)))
		}
		size := len(content)
		w.Header().Add("Content-Length", strconv.Itoa(size))
		w.Header().Add("Last-Modified", lastModified)

		if writeBody { //this is a render or a template, not a real file
			w.Write(content)
		}
		return
	}

	//file assured not to be a directory
	//also not a render of the dir
	//this is an actual file

	{ //mime-type detection
		DetectBuff := make([]byte, 512)
		n, err := File.ReadAt(DetectBuff, 0)
		//fmt.Println("N:::", n)
		if err != nil {
			if err != io.EOF {
				w.WriteHeader(http.StatusInternalServerError)
				vPrintf(1, "[%d] error reading file: %v\n", id, err)
				return
			}
		}
		MimeType = mime.TypeByExtension("." + Extension(File.Name()))
		//fmt.Printf("(%s) mime: %s", File.Name(), MimeType)
		if MimeType == "" {
			MimeType = http.DetectContentType(DetectBuff[:n])
		}
	}

	{ //setting headers
		w.Header().Add("Content-Length", strconv.Itoa(int(fileInfo.Size())))
		w.Header().Add("Content-Type", MimeType)
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	}

	NumBytesToCopy := fileInfo.Size()
	{ // checking if request is partial content request
		if Range := r.Header.Get("Range"); Range != "" { //range request
			var (
				offset      int64
				higherBound int64
				n           int
			)
			n, _ = fmt.Sscanf(Range, "bytes=%d-%d", &offset, &higherBound)
			if n < 2 {
				higherBound = fileInfo.Size() - 1
			}
			//vPrintf(0, "%d, %d, %d, %d\n", offset, higherBound, fileInfo.Size(), n)
			{ //range verification
				if offset >= fileInfo.Size() {
					http.Error(w, "invalid range offset >= fileInfo.Size()", http.StatusRequestedRangeNotSatisfiable)
					return
				}
				if higherBound >= fileInfo.Size() {
					http.Error(w, "invalid range higherBound >= fileInfo.Size()", http.StatusRequestedRangeNotSatisfiable)
					return
				}
				if offset >= higherBound {
					http.Error(w, "invalid range offset >= higherBound", http.StatusRequestedRangeNotSatisfiable)
					return
				}
			}

			File.Seek(offset, 0)
			NumBytesToCopy = (higherBound - offset) + 1
			//vPrintf(0, "%d\n", NumBytesToCopy)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", NumBytesToCopy))
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, higherBound, fileInfo.Size()))
			w.WriteHeader(http.StatusPartialContent)
			vPrintf(3, "resp:				%v\n", w.Header())
		}
	}

	//vPrintf(1, "accept-ranges:   %s\n", w.Header().Get("Accept-Ranges"))
	if writeBody {
		_, err = io.CopyN(w, File, NumBytesToCopy)
		if err != nil {
			dPrintln("error io.Copy-ing:", err)
		}
	}
}

//ManageHEAD responds to Head requests by telling ManageGet to not actually write content
func (m *mainHandler) ManageHEAD(w http.ResponseWriter, r *http.Request) {
	m.ManageGET(w, r, false)
}

//ManagePOST manages post forms (currently only used in fs mode)
func (m *mainHandler) ManagePOST(w http.ResponseWriter, r *http.Request) { //TODO: plus tard.

	w.WriteHeader(http.StatusNotImplemented)

	//Le faire marcher
	//fmt.Println("about to have color")
	//fmt.Println(Colorize(fmt.Sprint(r.URL), ColorMagenta))
	//fmt.Println(Colorize(fmt.Sprintf("%s", RenderHeader(&r.Header)), ColorBlue))
	//fmt.Println("<body>")
	io.Copy(os.Stdout, r.Body)
	return
	fmt.Println("</body>")
	if mode != "fileserver" {
		return
	}
	var (
		//content []byte
		files [][]byte
	)
	t := r.Header.Get("Content-Type")
	vals := strings.Split(t, "; ")

	if vals[0] == "multipart/form-data" {
		var separator string
		fmt.Sscanf(vals[1], "boundary=%s", &separator)
		dPrintln("separator:", separator)
		//content = make([]byte, r.ContentLength)
		iPrintln("START")
		/*n, err := r.Body.Read(content)
		if int64(n) != r.ContentLength {
			wPrintln("ManagePost: couldn't read entire form:", err.Error())
			return
		}
		*/
		use(files)
		for i := 0; true; i++ {
			c, err := ReadSlice(r.Body, []byte("--"+separator))
			dPrintln("CCC:", string(c))
			if err != nil {
				wPrintln(err)
				if err == io.EOF {
					break
				}
			}
			if len(c) == 0 {
				continue
			}
			files = append(files, c)

		}
		//fmt.Fscanf(r.Body, "%s"+separator)
		iPrintln("DONE")
		fmt.Println(files)
		f, err := ParseFormFile(files...)
		if err != nil {
			Fatal(err)
		}
		for _, v := range f {
			file, err := os.Create(WorkingDir + r.URL.Path + "/" + v.FileName)
			if err != nil {
				wPrintln(err)
				continue
			}
			defer file.Close()
			file.Write(v.Data)
		}
	}
	dPrintln(t)
	iPrintln("GOT a Post request!!!")
	dPrintf("%+#v\n", r)

}

func render(base, folderPath string) ([]byte, error) { //simple rendu d'un dossier
	/*
		<a href="/autre dossier ou fichier></a>"
		...
		...
	*/
	out := "" //on a besoin de base pour dire à quoi correspond le / (root)
	folderstats, err := ioutil.ReadDir(base + folderPath)
	if err != nil {
		return nil, err
	}
	out += fmt.Sprintf("<pre>\n")
	for _, v := range folderstats {
		out += fmt.Sprintf("<a href=\"%s\">%s</a>\n", path.Clean(folderPath+"/"+v.Name()), v.Name())
	}
	out += fmt.Sprintf("</pre>\n")
	//out = fmt.Sprintf(BasicHTMLFile(), "", out)
	return []byte(out), nil
}

//BasicHTMLFile is a template for a very simple html webpage
func BasicHTMLFile() string {
	return `<!DOCTYPE html>
<html>
<head>%s</head>
<body>%s</body>
</html>`
}

//BasicFileServerHeader returns a template for posting forms in the fs mode
func BasicFileServerHeader() string {
	return form
}

const form = `
<form method="post" enctype="multipart/form-data">
<input type="file" id="fileid" name="filename"/>
<button>Submit</button>
</form>
`

//GetAddress is a simple way to get main ip address
func GetAddress() (addrs []net.IP) {
	o := make([]net.Addr, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		vPrintf(1, err.Error())
		return nil
	}
	dPrintln(ifaces)
	for i := range ifaces {
		if ifaces[i].Flags&net.FlagLoopback == 0 {
			addrs, err := ifaces[i].Addrs()
			if err != nil {
				vPrintf(1, err.Error())
				continue
			}
			if len(addrs) == 0 {
				continue
			}
			dPrintf("addresses of interface %d: %v\n", i, addrs)
			o = append(o, addrs...)

		}
	}
	if len(o) == 0 {
		//log.Printf("address: %s\n", addr)
	} else {
		for _, v := range o {
			//fmt.Println(v)
			ip := net.ParseIP(strings.Split(v.String(), "/")[0])
			addrs = append(addrs, ip)
			vPrintf(1, "other address: %v\n", ip)

		}
	}
	return

}

func Line(skip ...int) string { //tells line number
	var s int
	if len(skip) == 0 {
		s = 1
	} else {

		s = skip[0]
	}
	_, file, a, _ := runtime.Caller(s)
	split := strings.Split(file, "/")
	file = split[len(split)-1]
	return file + " " + strconv.Itoa(a)
}

func use(x ...interface{}) {

}
func ReadSlice(r io.Reader, delim []byte) ([]byte, error) {
	var (
		iDelim    int
		out       []byte = make([]byte, 0, len(delim))
		middleMan []byte = make([]byte, 0)
		oneByte          = make([]byte, 1)
	)

	for iDelim < len(delim) {
		//dPrintln("i", iDelim, string(out))
		_, err := r.Read(oneByte)
		if err != nil {
			return out, err
		}
		if oneByte[0] != delim[iDelim] {
			out = append(out, middleMan...)
			middleMan = make([]byte, 0)
			out = append(out, oneByte[0])
			iDelim = 0
			continue
		}
		middleMan = append(middleMan, oneByte[0])
		iDelim++
	}
	return out, nil
}

type FormFile struct {
	Data           []byte
	FileName       string
	WantedFileName string
}

func ParseFormFile(x ...[]byte) ([]*FormFile, error) {
	o := make([]*FormFile, 0)
	for _, v := range x {
		next := new(FormFile)
		var (
			FileName string
			Content  []byte
		)
		b := bytes.NewReader(v)
		f, err := ReadSlice(b, []byte("\r\n\r\n"))
		if err != nil {
			return nil, err
		}
		header := string(f)
		headerMap := ParseHeader(header)
		s, ok := headerMap["Content-Disposition"]
		if !ok {
			ePrintln(v, headerMap)
		}
		split := strings.Split(s, "; ")
		m := map[string]string{}
		for _, v := range split {
			duo := strings.Split(v, "=")
			if len(duo) != 2 {
				continue
			}
			m[duo[0]] = strings.TrimPrefix(strings.TrimSuffix(m[duo[1]], "\""), "\"")
		}
		FileName = m["filename"]
		Content, err = ioutil.ReadAll(b)
		if err != nil {
			return nil, err
		}
		next.Data = Content
		next.FileName = FileName
		o = append(o, next)
	}
	return o, nil
}

func ParseHeader(x string) map[string]string {
	o := map[string]string{}

	for _, v := range strings.Split(x, "\r\n") {
		if v == "" {
			continue
		}
		split := strings.Split(v, ": ")
		o[split[0]] = split[1]
	}

	return o
}

func ReadPostRequest(x string) map[string]string {
	o := make(map[string]string)
	split := strings.Split(x, "&")
	for _, v := range split {
		pair := strings.Split(v, "=")
		o[pair[0]] = pair[1]
	}
	return o
}
func WaitXInterrupt(x int, c chan os.Signal) chan struct{} {
	ret := make(chan struct{})
	go func() {
		for i := x; i >= 0; i-- {
			<-c
			iPrintf("\n%d interrupts remaining before force shutdown\n", i)
		}
		ret <- struct{}{}
	}()
	return ret
}

func waitTillDone(f func() error) chan error {
	o := make(chan error)
	go func() {
		o <- f()
	}()
	return o
}

func ManageServer(server *http.Server) chan int {
	done := make(chan int)
	go func(ret chan int) {
		channel := make(chan os.Signal)
		signal.Notify(channel, os.Interrupt)
		<-channel
		dPrintln("interrupt")
		fmt.Printf("server shutting down in %v\n", shutdowmTimeout.String())
		ctx, _ := context.WithTimeout(context.Background(), shutdowmTimeout)
		select {
		case <-WaitXInterrupt(10, channel):
			iPrintln("server shutdown forcefully")
		case err := <-waitTillDone(func() error { return server.Shutdown(ctx) }):
			if err != nil {
				dPrintln(err)
				ePrintln("server shutdown forcefully")
			} else {
				iPrintln("server shutdown cleanly")

			}
		}
		ret <- 0
	}(done)
	return done
}

func Flags() {
	{ //flag declaring and parsing
		{ //on demande le help, commit ou version
			flag.BoolVar(&isGitCommit, "commit", false, "prints the git commit the code was compiled on")
		}
		{ //standard
			//port number
			flag.StringVar(&port, "port", "8080", "defines the TCP port on which the web server is going to serve (must be a valid port number)") // le port (par défault c'est le 8080)

			//working dir
			flag.StringVar(&WorkingDir, "dir", defaultWorkingDir, "defines the directory the server is goig to serve") //le scope du serveur(les fichiers qui seront servit)
			//	 par défault c'est le fichier duquel le programme a été commencé

		}

		{ //log/info/debugging/on veut du text de couleur
			flag.BoolVar(&isVerbose, "v", false, "make the program more verbose")               //si on dit pleins d'informations
			flag.IntVar(&verbosityLevel, "V", 0, "sets the degree of verbosity of the program") //le niveau d'information plus ou moins utiles que l'on dit

			flag.BoolVar(&isDebug, "D", false, "used to show internal values usefule for debuging")

			flag.BoolVar(&isTellTime, "telltime", false, "each log entry will tell the time it was printed")
		}
		{ //HTTPS
			flag.BoolVar(&isTLS, "tls", false, "enables tls encryption (not yet implemented)") //si on utilise une encryption
			flag.StringVar(&PathCertFile, "certfile", "", "the location of the TLS certificate file")
			flag.StringVar(&PathKeyFile, "keyfile", "", "the location of the TLS key file")
		}
		{ //server specific := time.Unix(1<<63-1, 0)
			flag.BoolVar(&isKeepAliveEnabled, "A", true, "enables http keep-alives")
			flag.DurationVar(&shutdowmTimeout, "shutdown-timeout", time.Second*10, "time the server waits for current connections when shutting down")
			flag.DurationVar(&requestTimeout, "request-timeout", MaxDuration, "the time the server will wait for the request")
			flag.StringVar(&mode, "mode", "", "sets server mode")
			{ // web ui flags
				flag.BoolVar(&isWebUIEnabled, "webui", false, "enables web ui")
				flag.IntVar(&webUIport, "uiport", 8080, "specifies web ui port")
			}
		}

		flag.BoolVar(&isColorEnabled, "color", true, "enables or disables color in terminal log")

		{ //auth flags
			flag.BoolVar(&isAuthEnabled, "auth", false, "enable password access")
			flag.StringVar(&authPassword, "p", "", "sets the required password when authentification is enabled")
			flag.StringVar(&authUsername, "u", "", "sets the required password when authentification is enabled")
		}

		{ // file server (mode is fileServer)

		}

		flag.Parse() //on interprète
		if isGitCommit {
			fmt.Println(gitCommit)
			os.Exit(0)
		}
	}
}

func Fprint(w io.Writer, x ...interface{}) (int, error) {
	if isTellTime {
		return fmt.Fprint(w, time.Now().Format("Jan 2 15:04:05 MST 2006"), fmt.Sprint(x...))
	}
	return fmt.Fprint(w, x...)
}
func TextColor(colorCode int) string {
	return fmt.Sprintf("\033[3%dm", colorCode)
}

func TextStyle(styleCode int) string {
	return fmt.Sprintf("\033[%dm", styleCode)
}

func TextReset() string {
	return fmt.Sprintf("\033[%dm", ColorReset)
}

func Fatal(x interface{}) {
	ePrintln(x)
	os.Exit(1)
	//panic(x)not pretty to the user lol
}

func iPrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[INFOS]", ColorGreen), fmt.Sprint(a...)))
}

func iPrintln(a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintln(a...))
}

func iPrintf(format string, a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintf(format, a...))
}

func ePrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[ERROR]", ColorRed), fmt.Sprint(a...)))
}

func ePrintln(a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintln(a...))
}

func ePrintf(format string, a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintf(format, a...))
}

func wPrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[WARN ]", ColorYellow), fmt.Sprint(a...)))
}

func wPrintln(a ...interface{}) (int, error) {
	return wPrint(fmt.Sprintln(a...))
}

func wPrintf(format string, a ...interface{}) (int, error) {
	return wPrint(fmt.Sprintf(format, a...))
}

func vPrint(verbosityTreshold int, x ...interface{}) (int, error) {
	if verbosityLevel < verbosityTreshold {
		return 0, nil
	}

	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[VERBO]", ColorGrey), fmt.Sprint(x...)))

}

func vPrintf(verbosityTreshold int, f string, a ...interface{}) (int, error) { //log des choses selon le degré de verbosité

	return vPrint(verbosityTreshold, fmt.Sprintf(f, a...))
}

func vPrintln(t int, a ...interface{}) (int, error) {
	return vPrint(t, a...)
}

func Colorize(x string, code int) string {
	if isColorEnabled {
		return TextColor(code) + x + TextReset()
	}
	return x
}

func dPrintln(x ...interface{}) (int, error) {
	if !isDebug {
		return 0, nil
	}
	return dPrint(fmt.Sprintln(x...))
}

func dPrint(x ...interface{}) (int, error) {

	if !isDebug {
		return 0, nil
	}
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[DEBUG]", ColorCyan), fmt.Sprint(x...)))
}

func dPrintf(format string, x ...interface{}) (int, error) {
	if !isDebug {
		return 0, nil
	}
	return dPrint(fmt.Sprintf(format, x...))
}

func CheckAuth(w http.ResponseWriter, r *http.Request) error {

	w.Header().Add("WWW-Authenticate", "Basic") //add new login schemes

	var (
		password string
		username string
		err      error
		result   []byte
	)
	var (
		authScheme string
		s          string
	)
	total := r.Header.Get("Authorization") //checking if client knows he needs auth
	if total == "" {                       //he doesn't know
		return ErrorUnauthorized
	}

	_, err = fmt.Sscanf(total, "%s %s", &authScheme, &s)
	if err != nil {
		dPrintln("error scanning auth token:", err)

		return ErrorUnauthorized
	}

	result, err = base64.StdEncoding.DecodeString(s)
	if err != nil {
		fmt.Println(err)
		return ErrorUnauthorized
	}

	split := strings.Split(string(result), ":")
	username, password = split[0], split[1]
	dPrintf("username: %s password: %s\n", username, password)
	if password != authPassword || username != authUsername {
		return ErrorUnauthorized
	}
	return nil
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

func ParseCommandLine(s string) []string {
	var (
		outputSlice = make([]string, 0)
		currentWord []rune

		doubleQuoted bool
		quoted       bool
		escaped      bool
	)
	currentWord = make([]rune, 0, 32)
	for _, char := range s { //ranging through each char
		//fmt.Printf("char: %c\n", char)
		//fmt.Println(string(currentWord))
		if escaped { // if char before was \ :
			currentWord = append(currentWord, char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '\'' {
			quoted = !quoted
			continue
		}

		if char == '"' {
			doubleQuoted = !doubleQuoted
			continue
		}

		if char == ' ' {
			outputSlice = append(outputSlice, string(currentWord))
			currentWord = make([]rune, 0, 32)
			//fmt.Println(outputSlice)
			continue
		}
		currentWord = append(currentWord, char)

	}
	if len(currentWord) > 0 {
		outputSlice = append(outputSlice, string(currentWord))
	}
	outputSlice = StripBlankStrings(outputSlice)

	return outputSlice
}

func StripBlankStrings(s []string) []string {
	o := make([]string, len(s))
	for i := range s {
		if len(s[i]) == 0 {
			continue
		}
		o = append(o, s[i])
	}
	return o
}

func Extension(s string) string {
	a := strings.Split(s, ".")
	return a[len(a)-1]
}
