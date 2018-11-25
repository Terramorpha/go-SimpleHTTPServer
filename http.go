package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

//CheckAuth checks (if simple authentication is enabled) if requests is allowed to access content
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

	if password != MainConfig.Get("AuthPassword").String() {
		return ErrorUnauthorized
	}
	if username != MainConfig.Get("AuthUsername").String() {
		return ErrorUnauthorized
	}
	return nil
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
func BasicHTMLFile(head, body, script string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>%s</head>
<body>
%s
<script src="%s"></script>
</body>
</html>`, head, body, script)
}

//BasicFileServerHeader returns a template for posting forms in the fs mode
func BasicFileServerHeader() string {
	return form
}

const form = `
<form method="post" enctype="multipart/form-data">
<input type="file" id="fileid" name="filename" multiple>
<input type="submit" value="Submit">
</form>
`

//ManagePOST manages post forms (currently only used in fs mode)
func (m *mainHandler) ManagePOST(w http.ResponseWriter, r *http.Request) { //TODO: plus tard.
	if MainConfig.Get("Mode").String() != "fileserver" {
		dPrintln("got post, not correct mode:", MainConfig.Get("Mode").String())
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	t := r.Header.Get("Content-Type")
	dPrintf("header: %v\n", r.Header)
	dPrintf("content-type: \"%v\"\n", t)
	separator := strings.Split(r.Header.Get("Content-Type"), "; ")[1]
	separator = strings.Split(separator, "=")[1]

	multiReader := multipart.NewReader(r.Body, separator)
	dPrintln("maybe gonna print the shit")
	for {
		part, err := multiReader.NextPart()
		if err != nil {
			if err != io.EOF {
				wPrintln(err)
			}
			break
		}
		defer part.Close()
		dPrintf("printing the shit called %s\n", part.FileName())
		dPrintf("wd: %s\n", MainConfig.Get("WorkingDir").String())
		dPrintf("path: %s\n", r.URL.Path)
		dPrintf("filename: %s\n", part.FileName())
		tot := path.Clean(MainConfig.Get("WorkingDir").String() + r.URL.Path + "/" + part.FileName())
		dPrintf("total file path: %s\n", tot)

		f, err := os.Create(tot)
		if err != nil {
			panic(err)
		}
		io.Copy(f, part)

	}
	r.Method = "GET"
	m.ManageGET(w, r, true)

}

//ManageGET manages get requests.
//
//yeah.
//
//that's it
func (m *mainHandler) ManageGET(w http.ResponseWriter, r *http.Request, writeBody bool) { //serves get requests
	wd := MainConfig.Get("WorkingDir").String()
	{ //setting default header
		w.Header().Add("Accept-Ranges", "bytes")
	}

	var (
		MimeType string
		id       = m.NewRequest() // request id
	)

	if MainConfig.Get("IsAuth").Bool() { //basic auth
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
	if MainConfig.Get("IsDebug").Bool() {
		w.Header().Add("Request-Id", strconv.Itoa(id))
	}

	vPrintf(1, "[%d] asking for %v\n", id, r.URL.EscapedPath())

	ComposedPath := wd + path.Clean(r.URL.Path)
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
		switch MainConfig.Get("Mode").String() { // checks for index.html
		default:
			content, err = render(wd, r.URL.Path)
			if err != nil {
				http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
				vPrintf(1, "[%d] error rendering directory: %v\n", id, err)
				return
			}
			content = []byte(BasicHTMLFile("", string(content), ""))
		case "web":
			content, err = ioutil.ReadFile(ComposedPath + "/index.html")
			if err != nil {
				SendStatusFail(w, http.StatusNotFound)
				//http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
				return
			}
		case "fileserver":
			renderedFolder, err := render(wd, r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				vPrintf(2, "[%d] error reading index.html: %v\n", id, err)
				return
			}
			content = []byte(fmt.Sprintf(BasicHTMLFile("", BasicFileServerHeader()+string(renderedFolder), "")))
		}
		size := len(content)
		w.Header().Add("Content-Length", strconv.Itoa(size))
		w.Header().Add("Last-Modified", lastModified)

		if writeBody { //this is a render or a template, not a real file
			w.Write(content)
		}
		return
	}

	//file assured not to be a directorymultiple directory
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
			dPrintf("header for a range request: %v\n", r.Header)
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

//ServeHTTP separes different types of request.
func (m *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vPrintf(1, "new Connection: %s %s %s\n", r.Method, r.URL.EscapedPath(), r.Proto)
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
