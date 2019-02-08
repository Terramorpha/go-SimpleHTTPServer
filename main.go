package main

import (
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
	ColorBold = 1
	//ColorDim makes it dim
	ColorDim = 2
	//ColorUnderlined underlines
	ColorUnderlined = 4
	//ColorBlink makes the text blink
	ColorBlink = 5
	//ColorReversed I dunno
	ColorReversed = 7
	//ColorHidden hides text... completly useless in this particular context
	ColorHidden = 8

	//ColorBlack is black
	ColorBlack = 0
	//ColorRed is red
	ColorRed = 1
	//ColorGreen is Green
	ColorGreen = 2
	//ColorYellow is yellow
	ColorYellow = 3
	//ColorBlue is blue
	ColorBlue = 4
	//ColorPurple is purple
	ColorPurple = 5
	//ColorCyan is Cyan
	ColorCyan = 6
	//ColorWhite is white
	ColorWhite = 7
)

var (
	gitCommit = ""
)

var modes = []string{
	"web",
	"proxy",
}

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

/*
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
*/
var conf struct {
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
	isVersion          bool
}

func main() {

	if conf.isVerbose {
		iPrintf("verbosity level: %d\n", conf.verbosityLevel) //on dit le niveau de verbosité
		iPrintf("mode: %s\n", conf.mode)
	}

	if conf.isAuthEnabled {
		iPrintf("auth enabled\nUsername:%s\nPassword:%s\n", conf.authUsername, conf.authPassword)
	}
	han := new(mainHandler) //l'endroit où le serveur stockera ses variables tel que le nb de connections
	server := &http.Server{
		Addr:              ":" + conf.port,
		Handler:           han,
		ReadHeaderTimeout: conf.requestTimeout,
		ReadTimeout:       conf.requestTimeout,
		WriteTimeout:      conf.requestTimeout,
		IdleTimeout:       conf.requestTimeout,
	}
	server.SetKeepAlivesEnabled(conf.isKeepAliveEnabled)
	addrs := GetAddress()
	iPrintln("you can connect to this server on:")
	for _, v := range addrs {
		fmt.Printf("        "+"http://%s/\n", net.JoinHostPort(v.String(), strconv.Itoa(conf.portNum)))
	}
	if conf.isTLS { //l'encryption n'est pas implémentée, don si elle activée, crash
		//	Fatal(errors.New("tls not yet implemented"))
	}
	done := ManageServer(server) //manageserver permet de faire runner le server pi de savoir quand il est fermé
	//server.RegisterOnShutdown(func() {  }) //quoi faire quand le serveur ferme
	iPrintf("serving %s on port %s\n", conf.WorkingDir, conf.port)
	if conf.isTLS {
		//not yet implemented
		err := server.ListenAndServeTLS(conf.PathCertFile, conf.PathKeyFile)
		if err != http.ErrServerClosed {
			Fatal(err)
		}
	} else {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed { //sert les requêtes http sur le listener et le stockage choisi
			Fatal(err)
		}
	}
	<-done
}

func init() { //preparing server and checking
	Flags()

	{ //setting auth if pass or user is set
		if conf.authUsername != "" || conf.authPassword != "" {
			conf.isAuthEnabled = true
		}
	}
	//dPrintln(User)

	{ //checking mode string
		if conf.mode != "" {
			isIn := false
			for i := range modes {
				if modes[i] == conf.mode {
					isIn = true
					break
				}
			}
			if !isIn {
				Fatal("invalid mode. only these are valid:\n" + fmt.Sprint("web", "fileserver"))
			}
		}
	}
	{ //checking TLS settings
		if conf.isTLS {
			if conf.PathCertFile == "" {
				Fatal("path of certificate file must be given for TLS to work (-certfile)")
			}
			if conf.PathKeyFile == "" {
				Fatal("path of key file must be given for TLS to work (-keyfile)")
			}
			certfile, err := os.OpenFile(conf.PathCertFile, os.O_RDONLY, 0400)
			if err != nil {
				Fatal(fmt.Sprintf("certfile %s cant be accessed, TLS can't work", conf.PathCertFile))
			}
			certfile.Close()
			keyfile, err := os.OpenFile(conf.PathKeyFile, os.O_RDONLY, 0400)
			if err != nil {
				Fatal(fmt.Sprintf("keyfile %s cant be accessed, TLS can't work", conf.PathKeyFile))
			}
			keyfile.Close()
		}

	}
	{ //checking working directory exists (or else nothing will get done)
		file, err := os.Open(conf.WorkingDir)
		defer file.Close()
		if err != nil {

			switch {
			case os.IsNotExist(err):
				Fatal(fmt.Sprintf("directory %s doesn't exist: %v", conf.WorkingDir, err))
			case os.IsPermission(err):
				Fatal(fmt.Sprintf("you don't have permission to open %s directory: %v", conf.WorkingDir, err))
			case os.IsTimeout(err):
				Fatal(fmt.Sprintf("getting directory %s timed out: %v", conf.WorkingDir, err))
			}

		}
		if stat, err := file.Stat(); err != nil {
			Fatal(fmt.Sprintf("error getting file stats: %v", err))
		} else {
			if !stat.IsDir() {
				Fatal(fmt.Sprintf("%s is not a directory", conf.WorkingDir))
			}
		}

	}
	{ //port permission  && validity checking

		portnum, err := net.LookupPort("tcp", conf.port)
		if err != nil {

			Fatal("error invalid port number: " + err.Error())
		}
		conf.portNum = portnum
		if conf.portNum < 1024 {
			wPrintf("need root to bind on port %d\n", conf.portNum)
		}

	}

	{ //setting correct verbosity levels

		if conf.verbosityLevel > 0 { //au cas ou l'utilisateur a pensé à metre le niveau de verbosité mais pas d'activer la verbosité
			conf.isVerbose = true //on l'active
		}
		if conf.isVerbose && conf.verbosityLevel == 0 { //si c'est verbose, le niveau devrait être plus élevé que 0
			conf.verbosityLevel = 1
		}

	}
}

type mainHandler struct {
	//ReqCount is a simple tracker for the number of http Request mainHandler has received
	Requests int
}

//ServeHTTP separes different types of request.
func (m *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if conf.mode == "proxy" {
		if r.Method != "CONNECT" {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
	}
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

//ManageConnect esrablished a TCP tunnel between the requested server and the client
func (m *mainHandler) ManageCONNECT(w http.ResponseWriter, r *http.Request) { //currently shit
	dPrintf("%#+v\n", r.URL)
	connRemote, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	defer connRemote.Close()
	w.Header().Del("Content-Length")
	w.WriteHeader(http.StatusOK)

	hj, ok := w.(http.Hijacker)
	if !ok {
		wPrintf("Coulnd't create hijacker\n")
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		wPrintf("Coulnd't hijack connection: %s\n", err)
	}
	defer conn.Close()
	buf.Flush()

	go io.Copy(conn, connRemote)
	io.Copy(connRemote, conn)

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

	if conf.isAuthEnabled { //basic auth
		err := CheckAuth(w, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			vPrintf(1, "failed auth from %v: %s\n", r.RemoteAddr, err.Error())
			return
		}

	}

	iPrintf("%v requested %s\n", r.RemoteAddr, r.URL.Path)
	{ //logging
		log := fmt.Sprintf("[%d] got %s request to %v from %v\n", id, r.Method, r.RequestURI, r.RemoteAddr)
		finishedServing := fmt.Sprintf("finished serving [%d] at %v\n", id, r.RemoteAddr)
		vPrintf(1, log)
		defer vPrintf(1, finishedServing)
	}
	// the actual URL
	if conf.isDebug {
		w.Header().Add("Request-Id", strconv.Itoa(id))
	}

	vPrintf(1, "[%d] asking for %v\n", id, r.URL.EscapedPath())

	ComposedPath := conf.WorkingDir + path.Clean(r.URL.Path)
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
			lastModified = time.Now().UTC().Format(http.TimeFormat)
			content      []byte
		)
		switch conf.mode { // checks for index.html
		default:
			content, err = render(conf.WorkingDir, r.URL.Path)
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
			renderedFolder, err := render(conf.WorkingDir, r.URL.Path)
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
		MimeType = mime.TypeByExtension("." + path.Ext(File.Name()))
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
	//io.Copy(os.Stdout, r.Body)
	return
	fmt.Println("</body>")
	if conf.mode != "fileserver" {
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
			file, err := os.Create(conf.WorkingDir + r.URL.Path + "/" + v.FileName)
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

//Line returns the line number it is called from
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

//ReadSlice is Like bufio.ReadLine but you can provide a delimiter string
func ReadSlice(r io.Reader, delim []byte) ([]byte, error) {
	var (
		iDelim    int
		out       = make([]byte, 0, len(delim))
		middleMan = make([]byte, 0)
		oneByte   = make([]byte, 1)
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

//FormFile is a structure representing a form
type FormFile struct {
	Data           []byte
	FileName       string
	WantedFileName string
}

//ParseFormFile parses the form
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

//ParseHeader parses rhe header in a multipart form
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

//WaitXInterrupt waits for x interrupts before returning
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

//ManageServer manages the server and srops it whenever it receives SIGINT
func ManageServer(server *http.Server) chan int {
	done := make(chan int)
	go func(ret chan int) {
		channel := make(chan os.Signal)
		signal.Notify(channel, os.Interrupt)
		<-channel
		dPrintln("interrupt")
		fmt.Printf("server shutting down in %v\n", conf.shutdowmTimeout.String())
		ctx, _ := context.WithTimeout(context.Background(), conf.shutdowmTimeout)
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

//Flags creates all the flags
func Flags() {
	{ //flag declaring and parsing

		flag.BoolVar(&conf.isVersion, "version", false, "prints version information")
		{ //on demande le help, commit ou version
			flag.BoolVar(&conf.isGitCommit, "commit", false, "prints the git commit the code was compiled on")
		}
		{ //standard
			//port number
			flag.StringVar(&conf.port, "port", "8080", "defines the TCP port on which the web server is going to serve (must be a valid port number)") // le port (par défault c'est le 8080)

			//working dir
			flag.StringVar(&conf.WorkingDir, "dir", defaultWorkingDir, "defines the directory the server is goig to serve") //le scope du serveur(les fichiers qui seront servit)
			//	 par défault c'est le fichier duquel le programme a été commencé

		}

		{ //log/info/debugging/on veut du text de couleur
			flag.BoolVar(&conf.isVerbose, "v", false, "make the program more verbose")               //si on dit pleins d'informations
			flag.IntVar(&conf.verbosityLevel, "V", 0, "sets the degree of verbosity of the program") //le niveau d'information plus ou moins utiles que l'on dit

			flag.BoolVar(&conf.isDebug, "D", false, "used to show internal values usefule for debuging")

			flag.BoolVar(&conf.isTellTime, "telltime", false, "each log entry will tell the time it was printed")
		}
		{ //HTTPS
			flag.BoolVar(&conf.isTLS, "tls", false, "enables tls encryption (not yet implemented)") //si on utilise une encryption
			flag.StringVar(&conf.PathCertFile, "certfile", "", "the location of the TLS certificate file")
			flag.StringVar(&conf.PathKeyFile, "keyfile", "", "the location of the TLS key file")
		}
		{ //server specific := time.Unix(1<<63-1, 0)
			flag.BoolVar(&conf.isKeepAliveEnabled, "A", true, "enables http keep-alives")
			flag.DurationVar(&conf.shutdowmTimeout, "shutdown-timeout", time.Second*10, "time the server waits for current connections when shutting down")
			flag.DurationVar(&conf.requestTimeout, "request-timeout", MaxDuration, "the time the server will wait for the request")
			flag.StringVar(&conf.mode, "mode", "", "sets server mode")
			{ // web ui flags
				flag.BoolVar(&conf.isWebUIEnabled, "webui", false, "enables web ui")
				flag.IntVar(&conf.webUIport, "uiport", 8080, "specifies web ui port")
			}
		}

		flag.BoolVar(&conf.isColorEnabled, "color", true, "enables or disables color in terminal log")

		{ //auth flags
			flag.BoolVar(&conf.isAuthEnabled, "auth", false, "enable password access")
			flag.StringVar(&conf.authPassword, "p", "", "sets the required password when authentification is enabled")
			flag.StringVar(&conf.authUsername, "u", "", "sets the required password when authentification is enabled")
		}

		{ // file server (mode is fileServer)

		}

		flag.Parse() //on interprète
		if conf.isVersion {
			fmt.Fprintf(os.Stderr,
				"git commit: %s\ncpu architecture: %s\noperation system: %s\n",
				gitCommit,
				runtime.GOARCH,
				runtime.GOOS,
			)
			os.Exit(0)
		}
		if conf.isGitCommit {
			fmt.Println(gitCommit)
			os.Exit(0)
		}
	}
}

//Fprint prints is the bottom function of all the log functions and prepends the time whenever needed
func Fprint(w io.Writer, x ...interface{}) (int, error) {
	if conf.isTellTime {
		return fmt.Fprint(w, time.Now().Format("Jan 2 15:04:05 MST 2006"), fmt.Sprint(x...))
	}
	return fmt.Fprint(w, x...)
}

//TextColor creates escap sequence from color
func TextColor(colorCode int) string {
	return fmt.Sprintf("\033[3%dm", colorCode)
}

//TextStyle creates escape sequence from style
func TextStyle(styleCode int) string {
	return fmt.Sprintf("\033[%dm", styleCode)
}

//TextReset creates escape sequence for resetting color and style
func TextReset() string {
	return fmt.Sprintf("\033[%dm", ColorReset)
}

//Fatal is a wrapper for when something doesn't work
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
	if conf.verbosityLevel < verbosityTreshold {
		return 0, nil
	}

	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[VERBO]", ColorWhite), fmt.Sprint(x...)))

}

func vPrintf(verbosityTreshold int, f string, a ...interface{}) (int, error) { //log des choses selon le degré de verbosité

	return vPrint(verbosityTreshold, fmt.Sprintf(f, a...))
}

func vPrintln(t int, a ...interface{}) (int, error) {
	return vPrint(t, a...)
}

//Colorize colorizes the given string
func Colorize(x string, code int) string {
	if conf.isColorEnabled {
		return TextColor(code) + x + TextReset()
	}
	return x
}

func dPrintln(x ...interface{}) (int, error) {
	if !conf.isDebug {
		return 0, nil
	}
	return dPrint(fmt.Sprintln(x...))
}

func dPrint(x ...interface{}) (int, error) {

	if !conf.isDebug {
		return 0, nil
	}
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[DEBUG]", ColorCyan), fmt.Sprint(x...)))
}

func dPrintf(format string, x ...interface{}) (int, error) {
	if !conf.isDebug {
		return 0, nil
	}
	return dPrint(fmt.Sprintf(format, x...))
}

//CheckAuth verifies the authentication field is correct
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
	if password != conf.authPassword || username != conf.authUsername {
		return ErrorUnauthorized
	}
	return nil
}

//RenderHeader turns the header back into text
func RenderHeader(h *http.Header) string {
	o := ""
	for i, v := range *h {
		o += fmt.Sprintf("%s: %v\n", i, v)
	}
	return o

}

//SendStatusFail fails rhe request
func SendStatusFail(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	fmt.Fprintln(w, code, http.StatusText(code))
}

//Extens
func Extension(s string) string {
	a := strings.Split(s, ".")
	return a[len(a)-1]
}
