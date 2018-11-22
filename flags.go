package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func GetFlags() {
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
			flag.BoolVar(&isKeepAlive, "A", true, "enables http keep-alives")
			flag.DurationVar(&shutdowmTimeout, "shutdown-timeout", time.Second*10, "time the server waits for current connections when shutting down")
			flag.DurationVar(&requestTimeout, "request-timeout", MaxDuration, "the time the server will wait for the request")
			flag.StringVar(&mode, "mode", "", "sets server mode")
			{ // web ui flags
				flag.BoolVar(&isWebUI, "webui", false, "enables web ui")
				flag.IntVar(&webUIport, "uiport", 8080, "specifies web ui port")
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
	}
}

func CheckFlags() {
	{ //setting auth if pass or user is set
		if authUsername != "" || authPassword != "" {
			isAuth = true
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
