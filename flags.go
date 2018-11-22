package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var MainConfig = Config{
	&Option{
		Name:  "Port",
		Type:  "port",
		Value: "",
	},
	&Option{
		Name:  "WorkingDir",
		Type:  "file",
		Value: "",
	},
	&Option{
		Name:  "PathCertFile",
		Type:  "file",
		Value: "",
	},
	&Option{
		Name:  "PathKeyFile",
		Type:  "file",
		Value: "",
	},
	&Option{
		Name:  "Mode",
		Type:  "string",
		Value: "",
	},
	&Option{
		Name:  "ShutdownTimeout",
		Type:  "duration",
		Value: "",
	},
	&Option{
		Name:  "RequestTimeout",
		Type:  "duration",
		Value: "",
	},
	&Option{
		Name:  "AuthPassword",
		Type:  "string",
		Value: "",
	},
	&Option{
		Name:  "AuthUsername",
		Type:  "string",
		Value: "",
	},
	&Option{
		Name:  "IsTLS",
		Type:  "bool",
		Value: "",
	},
	&Option{
		Name:  "IsKeepAlive",
		Type:  "bool",
		Value: "",
	},
	&Option{
		Name:  "IsAuth",
		Type:  "bool",
		Value: "",
	},
}

func GetFlags() {

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
		webUIport    string

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
	//flag declaring and parsing
	//on demande le help, commit ou version
	flag.BoolVar(&isGitCommit, "commit", false, "prints the git commit the code was compiled on")

	{
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
		flag.BoolVar(&isKeepAlive, "A", true, "enables http keep-alives")
		flag.DurationVar(&shutdowmTimeout, "shutdown-timeout", time.Second*10, "time the server waits for current connections when shutting down")
		flag.DurationVar(&requestTimeout, "request-timeout", MaxDuration, "the time the server will wait for the request")
		flag.StringVar(&mode, "mode", "", "sets server mode")
		{ // web ui flags
			flag.BoolVar(&isWebUI, "webui", false, "enables web ui")
			flag.StringVar(&webUIport, "uiport", "8080", "specifies web ui port")
		}
	}

	flag.BoolVar(&isColored, "color", true, "enables or disables color in terminal log")

	{ //auth flags
		flag.BoolVar(&isAuth, "auth", false, "enable password access")
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

	{ //putting it in the string object #01

		MainConfig.Set("Port", port)

		//WorkingDir is the root of the server
		MainConfig.Set("WorkingDir", WorkingDir)
		//PathCertFile is the file from which http.ListenAndServeTLS will get its certificates
		MainConfig.Set("PathCertFile", PathCertFile)
		//PathKeyFile is the file from which http.ListenAndServeTLS will get its encryption keys
		MainConfig.Set("PathKeyFile", PathKeyFile)

		MainConfig.Set("VerbosityLevel", PathCertFile)

		MainConfig.Set("Mode", mode)
		MainConfig.Set("ShutdownTimeout", shutdowmTimeout.String())
		MainConfig.Set("RequestTimeout", requestTimeout.String())

		MainConfig.Set("AuthPassword", authPassword)
		MainConfig.Set("AuthUserName", authUsername)
		MainConfig.Set("WebUIPort", webUIport)

		MainConfig.Set("IsVerbose", strconv.FormatBool(isVerbose))
		MainConfig.Set("IsTLS", strconv.FormatBool(isTLS))
		MainConfig.Set("IsKeepAlive", strconv.FormatBool(isKeepAlive))
		MainConfig.Set("IsDebug", strconv.FormatBool(isDebug))
		MainConfig.Set("IsAuth", strconv.FormatBool(isAuth))
		MainConfig.Set("IsTellTime", strconv.FormatBool(isTellTime))

		MainConfig.Set("IsWebUI", strconv.FormatBool(isWebUI))

		MainConfig.Set("IsColored", strconv.FormatBool(isColored))
	}
	fmt.Printf("%#+v\n", MainConfig)
}

func CheckFlags() {
	{ //setting auth if pass or user is set
		if MainConfig.Get("AuthUserName") != "" || MainConfig.Get("AuthPassword") != "" {
			MainConfig.Set("IsAuth", "true")
		}
	}
	//dPrintln(User)

	{ //checking mode string
		if MainConfig.Get("Mode") != "" {
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
