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
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var ( //where we put flag variables and global variables
	//User contains all information about the user running the program (currently only used to check if program is run as root)
	User    *user.User
	port    string
	portNum int
	//WorkingDir represent the root of the server
	WorkingDir         = "./"
	isVerbose          = false
	isTLS              bool
	verbosityLevel     int
	isKeepAliveEnabled bool
	isDebug            bool
	mode               string
	shutdowmTimeout    time.Duration

	isAuthEnabled bool
	authPassword  string
	authUsername  string

	isWebUIEnabled bool
	webUIPort      int

	isColorEnabled bool
)

func init() { //preparing server and checking
	Flags()

	{ //setting auth if pass or user is set
		if authUsername != "" || authPassword != "" {
			isAuthEnabled = true
		}
	}
	//dPrintln(User)

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
	{

		if verbosityLevel > 0 { //au cas ou l'utilisateur a pensé à metre le niveau de verbosité mais pas d'activer la verbosité
			isVerbose = true //on l'active
		}
		if isVerbose && verbosityLevel == 0 { //si c'est verbose, le niveau devrait être plus élevé que 0
			verbosityLevel = 1
		}

	}
}

func main() {
	if isVerbose {
		iPrintf("verbosity level: %d\n", verbosityLevel) //on dit le niveau de verbosité
		iPrintf("mode: %s\n", mode)
	}

	if isAuthEnabled {
		iPrintf("auth enabled\nUsername:%s\nPassword:%s\n", authUsername, authPassword)
	}
	han := new(mainHandler) //l'endroit où le serveur stockera ses variables tel que le nb de connections
	server := &http.Server{Addr: ":" + port, Handler: han}
	server.SetKeepAlivesEnabled(isKeepAliveEnabled)
	addrs := GetAddress()
	iPrintln("you can connect to this server on:")
	for _, v := range addrs {
		fmt.Printf("http://%s/\n", net.JoinHostPort(v.String(), strconv.Itoa(portNum)))
	}
	if isTLS { //l'encryption n'est pas implémentée, don si elle activée, crash
		Fatal(errors.New("tls not yet implemented"))
	}
	if isWebUIEnabled {
		go WebUi()
	}
	done := ManageServer(server) //manageserver permet de faire runner le server pi de savoir quand il est fermé
	//server.RegisterOnShutdown(func() {  }) //quoi faire quand le serveur ferme
	iPrintf("serving %s on port %s\n", WorkingDir, port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed { //sert les requêtes http sur le listener et le stockage choisi
		Fatal(err)
	}
	<-done
}

type mainHandler struct {
	//ReqCount is a simple tracker for the number of http Request mainHandler has received
	Requests  int
	Succeeded int
}

func (m *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	switch r.Method {
	case "GET":
		m.ManageGET(w, r, true)
	case "HEAD":
		m.ManageHEAD(w, r)
	case "POST":
		m.ManagePOST(w, r)
	}

}

func (m *mainHandler) NewRequest() int {
	a := m.Requests
	m.Requests++
	return a
}

func iPrint(a ...interface{}) (int, error) {
	return fmt.Printf("%s %s", Colorize("[INFOS]", ColorGreen), fmt.Sprint(a...))
}

func iPrintln(a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintln(a...))
}

func iPrintf(format string, a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintf(format, a...))
}

func ePrint(a ...interface{}) (int, error) {
	return fmt.Printf("%s %s", Colorize("[ERROR]", ColorRed), fmt.Sprint(a...))
}

func ePrintln(a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintln(a...))
}

func ePrintf(format string, a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintf(format, a...))
}

func wPrint(a ...interface{}) (int, error) {
	return fmt.Printf("%s %s", Colorize("[WARN ]", ColorYellow), fmt.Sprint(a...))
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

	return fmt.Print(fmt.Sprintf("%s %s", Colorize("[VERBO]", ColorGrey), fmt.Sprint(x...)))

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

func ceil(x, y int) int {
	if x > y {
		return y
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
	return fmt.Print(fmt.Sprintf("%s %s", Colorize("[DEBUG]", ColorCyan), fmt.Sprint(x...)))
}

func dPrintf(format string, x ...interface{}) (int, error) {
	if !isDebug {
		return 0, nil
	}
	return dPrint(fmt.Sprintf(format, x...))
}

func (m *mainHandler) ManageGET(w http.ResponseWriter, r *http.Request, writeBody bool) {

	if isAuthEnabled {
		{ //basic auth
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
			total := r.Header.Get("Authorization")
			if total == "" {
				w.Header().Add("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			_, err = fmt.Sscanf(total, "%s %s", &authScheme, &s)
			if err != nil {
				fmt.Println(err)
				w.Header().Add("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			result, err = base64.StdEncoding.DecodeString(s)
			if err != nil {
				fmt.Println(err)
				w.Header().Add("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			split := strings.Split(string(result), ":")
			username, password = split[0], split[1]
			dPrintf("username: %s password: %s\n", username, password)
			if password != authPassword || username != authUsername {
				w.Header().Add("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
	}

	var (
		MimeType string
	)

	id := m.NewRequest() // request id
	{                    //logging

		vPrintf(1, "[%d] got %s request to %v from %v (%v)\n", id, r.Method, r.Host, r.RemoteAddr, r.RequestURI)
		vPrintf(2, "received request %v\n", *r)
		defer vPrintf(1, "finished serving [%d] at %v\n", id, r.RemoteAddr)
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dPrintf("file opened: %s\n", ComposedPath)
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
				http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
			}
		case "fileserver":
			renderedFolder, err := render(WorkingDir, r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
			}
			content = []byte(fmt.Sprintf(BasicHTMLFile(), "", BasicFileServerHeader()+string(renderedFolder)))
		}
		size := len(content)
		w.Header().Add("Content-Length", strconv.Itoa(size))
		w.Header().Add("Last-Modified", lastModified)

		if writeBody {
			w.Write(content)
		}
		return
	}

	//file assured not to be a directory
	//also not a render of the dir
	//this is an actual file

	{ //mime-type detection
		DetectBuff := make([]byte, 512)
		_, err = File.ReadAt(DetectBuff, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			vPrintf(1, "[%d] error reading file: %v\n", id, err)
			return
		}
		MimeType = http.DetectContentType(DetectBuff)
	}

	//setting headers
	{
		w.Header().Add("Content-Length", strconv.Itoa(int(fileInfo.Size())))
		w.Header().Add("Content-Type", MimeType)
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Add("Accept-Ranges", "bytes")
	}

	NumBytesToCopy := fileInfo.Size()
	{ // checking if request is partial content request
		if Range := r.Header.Get("Range"); Range != "" { //range request
			var (
				offset      int64
				higherBound int64
				n           int
			)
			n, err = fmt.Sscanf(Range, "bytes=%d-%d", &offset, &higherBound)
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
		io.CopyN(w, File, NumBytesToCopy)
	}
}

func (m *mainHandler) ManageHEAD(w http.ResponseWriter, r *http.Request) {
	m.ManageGET(w, r, false)
}

func (m *mainHandler) ManagePOST(w http.ResponseWriter, r *http.Request) {
	/*f, err := os.Create(strconv.Itoa(int(time.Now().Unix())))
	if err != nil {
		ePrintln(err)
		return
	}*/
	var (
		content []byte
		files   [][]byte
	)
	t := r.Header.Get("Content-Type")
	vals := strings.Split(t, "; ")

	if vals[0] == "multipart/form-data" {
		var separator string
		fmt.Sscanf(vals[1], "boundary=%s", &separator)
		dPrintln("separator:", separator)
		content = make([]byte, r.ContentLength)
		iPrintln("START")
		n, err := r.Body.Read(content)
		if int64(n) != r.ContentLength {
			wPrintln("ManagePost: couldn't read entire form:", err.Error())
			return
		}

		var (
			header    string
			headerMap map[string]string
		)
		b := bytes.NewReader()
		for {
			fmt.Fscanf(b, "")
		}
		//fmt.Fscanf(r.Body, "%s"+separator)
		iPrintln("DONE")
	}
	dPrintln(t)
	iPrintln("GOT a Post request!!!")
	dPrintf("%+#v\n", r)

}

func render(base, folderPath string) ([]byte, error) {
	out := ""
	folderstats, err := ioutil.ReadDir(base + folderPath)
	if err != nil {
		return []byte(out), err
	}
	out += fmt.Sprintf("<pre>\n")
	for _, v := range folderstats {
		out += fmt.Sprintf("<a href=\"%s\">%s</a>\n", path.Clean(folderPath+"/"+v.Name()), v.Name())
	}
	out += fmt.Sprintf("</pre>\n")
	//out = fmt.Sprintf(BasicHTMLFile(), "", out)
	return []byte(out), nil
}

func BasicHTMLFile() string {
	return `<!DOCTYPE html>
<html>
<head>%s</head>
<body>%s</body>
</html>`
}

func BasicFileServerHeader() string {
	return form
}

const form = `
<form method="post" enctype="multipart/form-data">
<input type="file" name="My file" id="file" multiple/>
<button>Submit</button>
</form>
`

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

func ParseDuration(s string) (time.Duration, error) {
	var char rune
	var num int
	n, _ := fmt.Sscanf(s, "%d%c")
	if n != 2 {
		return 0, errors.New("invalid duration format")
	}
	switch char {
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	case 's':
		return time.Duration(num) * time.Second, nil
	default:
		return 0, errors.New("invalid time unit")
	}
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

func Line(skip ...int) string {
	var s int
	if len(skip) == 0 {
		s = 1
	} else {

		s = skip[0]
	}
	_, file, a, _ := runtime.Caller(s)
	split := strings.Split(file, "/")
	file = split[len(split)-1]
	return file + strconv.Itoa(a)
}

func use(x ...interface{}) {

}

func WebUi() {
	var (
		err error
	)

	han := new(settingsHandler)
	han.wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(webUIPort))
	if err != nil {
		panic(err)
	}
	err = http.Serve(listener, han)
	if err != nil {
		panic(err)
	}
}

type settingsHandler struct {
	wsUpgrader websocket.Upgrader
}

func (s *settingsHandler) Log(x ...interface{}) {
	o := "[webUI] "
	o += fmt.Sprint(x...)
	fmt.Println(o)
}

func (s *settingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.Log("got GET request")
	}
	if strings.HasPrefix(r.RequestURI, "/ws/") { //websockets
		conn, err := s.wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			s.Log(err)
		}
		go s.ManageWebsocket(conn)

	}
}

type SettingsUpdate struct {
	Type    string
	Content map[string]string
}

func (s *settingsHandler) ManageWebsocket(c *websocket.Conn) {

}

const (
	ColorReset      = 0
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
	panic(x)
}

func Flags() {
	{ //flag declaring and parsing
		{ //standard
			//port number
			flag.StringVar(&port, "port", "8080", "defines the TCP port on which the web server is going to serve (must be a valid port number)") // le port (par défault c'est le 8080)

			//working dir
			flag.StringVar(&WorkingDir, "dir", "./", "defines the directory the server is goig to serve") //le scope du serveur(les fichiers qui seront servit)
			//	 par défault c'est le fichier duquel le programme a été commencé

		}

		{ //log/info/debugging/on veut du text de couleur
			flag.BoolVar(&isVerbose, "v", false, "make the program more verbose")               //si on dit pleins d'informations
			flag.IntVar(&verbosityLevel, "V", 0, "sets the degree of verbosity of the program") //le niveau d'information plus ou moins utiles que l'on dit

			flag.BoolVar(&isDebug, "D", false, "used to show internal values usefule for debuging")
		}

		goto Apres
		{ //patentes qui marchent pas
			flag.BoolVar(&isTLS, "tls", false, "enables tls encryption (not yet implemented)") //si on utilise une encryption

		}
	Apres:
		{ //server specific
			flag.BoolVar(&isKeepAliveEnabled, "A", false, "enables http keep-alives")
			flag.DurationVar(&shutdowmTimeout, "shutdown-timeout", time.Second*10, "time the server waits for current connections when shutting down")
			flag.StringVar(&mode, "mode", "", "sets server mode")
			{ // web ui flags
				flag.BoolVar(&isWebUIEnabled, "webui", false, "enables web ui")
				flag.IntVar(&webUIPort, "uiPort", 8080, "specifies web ui port")
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

	}
}
