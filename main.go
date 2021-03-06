// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/bitmark-inc/bitmark-webgui/api"
	"github.com/bitmark-inc/bitmark-webgui/configuration"
	"github.com/bitmark-inc/bitmark-webgui/fault"
	"github.com/bitmark-inc/bitmark-webgui/utils"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var GlobalConfig *configuration.Configuration
var BitmarkWebguiConfigFile string

var mainLog *logger.L
var bitmarkConsoleProxy *httputil.ReverseProxy

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	var configFile string

	app := cli.NewApp()
	app.Name = "bitmark-webgui"
	app.Usage = "Configuration program for bitmarkd"
	app.Version = Version()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config-file, c",
			Value:       "",
			Usage:       "*bitmark-webgui config file",
			Destination: &configFile,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "setup",
			Usage: "Initialise bitmark-webgui configuration",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "hostname, H",
					Value: "",
					Usage: "generate server certificate with the hostname [localhost]",
				},
				cli.StringFlag{
					Name:  "data-directory, d",
					Value: "",
					Usage: "the direcotry of web and log",
				},
				cli.StringFlag{
					Name:  "password, P",
					Value: "webgui",
					Usage: "the password for the service",
				},
			},
			Action: func(c *cli.Context) error {
				runSetup(c, configFile)
				return nil
			},
		},
		{
			Name:  "start",
			Usage: "start bitmark-webgui",
			Action: func(c *cli.Context) error {
				runStart(c, configFile)
				return nil
			},
		},
	}

	app.Run(os.Args)
}

func runSetup(c *cli.Context, configFile string) {

	// set data-directory
	dataDir := c.String("data-directory")
	if "" == dataDir {
		dataDir = filepath.Dir(configFile)
	}
	defaultConfig, err := configuration.GetDefaultConfiguration(dataDir)
	if nil != err {
		exitwithstatus.Message("Error: %v\n", err)
	}

	// set logger
	setupLogger(&defaultConfig.Logging)
	defer logger.Finalise()

	if nil != err {
		mainLog.Errorf("get config file path: %s error: %v", configFile, err)
		exitwithstatus.Message("Error: %v\n", err)
	}

	// Check if file exist
	if !utils.EnsureFileExists(configFile) {
		password := c.String("password")
		if "" == password {
			password = defaultConfig.Password
		}
		encryptPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if nil != err {
			mainLog.Errorf("Encrypt password failed: %v", err)
			exitwithstatus.Message("Error: %v\n", err)
		}
		defaultConfig.Password = string(encryptPassword)

		// generate config file
		err = configuration.UpdateConfiguration(configFile, defaultConfig)
		if nil != err {
			mainLog.Errorf("Generate config template failed: %v", err)
			exitwithstatus.Message("Error: %v\n", err)
		}
		mainLog.Info("Successfully setup bitmark-webgui configuration file")

		// gen certificate
		hostname := c.String("hostname")
		if "" != hostname {
			// gen certs
			cert, key, newCreate, err := utils.GetTLSCertFile(defaultConfig.DataDirectory)
			if nil != err {
				mainLog.Errorf("get TLS file failed: %v", err)
				exitwithstatus.Message("get TLS file failed: %v\n", err)
			}

			if newCreate {
				mainLog.Infof("Generate self signed certificate for hostname: %s", hostname)
				hostnames := []string{hostname}
				if err := utils.MakeSelfSignedCertificate("bitmark-webgui", cert, key, false, hostnames); nil != err {
					mainLog.Errorf("generate TLS file failed: %v", err)
					exitwithstatus.Message("generate TLS file failed: %v\n", err)
				}
			} else {
				mainLog.Error("TLS file existed")
				exitwithstatus.Message("TLS file existed\n")
			}
			mainLog.Info("Successfully generate TLS files")
		}
	} else {
		mainLog.Errorf("config file %s existed", configFile)
		exitwithstatus.Message("Error: %s existed\n", configFile)
	}

}

func runStart(c *cli.Context, configFile string) {

	if !utils.EnsureFileExists(configFile) {
		exitwithstatus.Message("Error: %v\n", fault.ErrNotFoundConfigFile)
	}

	BitmarkWebguiConfigFile = configFile

	// read bitmark-webgui config file
	if configs, err := configuration.GetConfiguration(configFile); nil != err {
		exitwithstatus.Message("Error: %v\n", err)
	} else {
		GlobalConfig = configs

		setupLogger(&configs.Logging)
		defer logger.Finalise()

		// initialise services
		if err := InitialiseService(configs); nil != err {
			mainLog.Criticalf("initialise background services failed: %v", err)
			exitwithstatus.Exit(1)
		}
		defer FinaliseBackgroundService()

		go func() {
			if err := startWebServer(GlobalConfig); err != nil {
				mainLog.Criticalf("%s", err)
				exitwithstatus.Message("Error: %v\n", err)
			}
		}()

		// turn Signals into channel messages
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		sig := <-ch
		mainLog.Infof("received signal: %v", sig)
		mainLog.Info("shutting down...")
	}
}

func setupLogger(logging *configuration.LoggerType) {
	// start logging
	if err := logger.Initialise(logging.File, logging.Size, logging.Count); nil != err {
		exitwithstatus.Message("%s: logger setup failed with error: %v", err)
	}

	logger.LoadLevels(logging.Levels)

	// create a logger channel for the main program
	mainLog = logger.New("main")
	mainLog.Info("starting…")
	mainLog.Debugf("loggerType: %v", logging)
}

func startWebServer(configs *configuration.Configuration) error {
	host := "0.0.0.0"
	port := strconv.Itoa(configs.Port)

	// serve web pages
	mainLog.Info("Set up server files")
	baseWebDir := path.Join(configs.DataDirectory, "/ui/public")
	http.Handle("/", http.FileServer(http.Dir(baseWebDir+"/")))

	// serve api
	mainLog.Info("Set up server api")
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/password", handleSetPassword)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/logout", handleLogout)
	http.HandleFunc("/api/bitcoind", handleBitcoind)
	http.HandleFunc("/api/bitmarkd", handleBitmarkd)
	http.HandleFunc("/api/prooferd", handleProoferd)

	// serve console proxy and web socket
	bitmarkConsoleUrlStr := "http://localhost:" + bitmarkConsoleService.Port()
	bitmarkConsoleUrl, err := url.Parse(bitmarkConsoleUrlStr)
	if err != nil {
		panic(err)
	}
	bitmarkConsoleProxy = httputil.NewSingleHostReverseProxy(bitmarkConsoleUrl)
	http.Handle("/console/ws", handleBitmarkConsoleWS())
	http.HandleFunc("/console/", handleBitmarkConsole)

	server := &http.Server{
		Addr:           host + ":" + port,
		ReadTimeout:    20 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if configs.EnableHttps {
		mainLog.Info("Starting https server...")
		// gen certs
		cert, key, newCreate, err := utils.GetTLSCertFile(configs.DataDirectory)
		if nil != err {
			return err
		}

		if newCreate {
			mainLog.Info("Generate self signed certificate...")
			if err := utils.MakeSelfSignedCertificate("bitmark-webgui", cert, key, false, nil); nil != err {
				return err
			}
		}

		if err := server.ListenAndServeTLS(cert, key); nil != err {
			return err
		}
	} else {
		mainLog.Info("Starting http server...")
		if err := server.ListenAndServe(); nil != err {
			return err
		}
	}
	// turn Signals into channel messages
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-ch
		mainLog.Infof("received signal: %v", sig)
	}()

	return nil
}

type webPagesConfigType struct {
	Host        string
	Port        string
	EnableHttps bool
}

func checkAuthorization(w http.ResponseWriter, req *http.Request, writeHeader bool, log *logger.L) bool {
	if GlobalConfig.EnableHttps {
		if err := api.GetAndCheckCookie(w, req, log); nil != err {
			log.Errorf("Error: %v", err)
			cookie := &http.Cookie{
				Name:   api.CookieName,
				Secure: true,
				MaxAge: -1,
			}
			http.SetCookie(w, cookie)
			if writeHeader {
				w.WriteHeader(http.StatusUnauthorized)
			}
			return false
		}
	}

	return true
}

func handleConfig(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-config")

	api.SetCORSHeader(w, req)

	switch req.Method {
	case `GET`: // list bitmark config
		if !checkAuthorization(w, req, true, log) {
			return
		}
		api.ListConfig(w, req, GlobalConfig.BitmarkConfigFile, GlobalConfig.ProoferdConfigFile, log)
	case `POST`:
		if !checkAuthorization(w, req, true, log) {
			return
		}
		api.UpdateConfig(w, req, GlobalConfig.BitmarkChain, GlobalConfig.BitmarkConfigFile, GlobalConfig.ProoferdConfigFile, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleSetPassword(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-bitmarkWebgui")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	switch req.Method {
	case `POST`:
		if !utils.EnsureFileExists(BitmarkWebguiConfigFile) {
			exitwithstatus.Message("Error: %s\n", fault.ErrNotFoundConfigFile)
		}
		if configs, err := configuration.GetConfiguration(BitmarkWebguiConfigFile); nil != err {
			exitwithstatus.Message("Error: %v\n", err)
		} else {
			GlobalConfig = configs
			api.SetBitmarkWebguiPassword(w, req, BitmarkWebguiConfigFile, GlobalConfig, log)
		}
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleLogin(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-login")
	api.SetCORSHeader(w, req)

	switch req.Method {
	case `GET`:
		if !checkAuthorization(w, req, true, log) {
			return
		}
		api.LoginStatus(w, GlobalConfig, log)
	case `POST`:
		if GlobalConfig.EnableHttps && checkAuthorization(w, req, false, log) {
			if err := api.WriteGlobalErrorResponse(w, fault.ApiErrAlreadyLoggedIn, log); nil != err {
				log.Errorf("Error: %v", err)
			}
			return
		}
		api.LoginBitmarkWebgui(w, req, GlobalConfig, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleLogout(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-logout")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	switch req.Method {
	case `POST`:
		api.LogoutBitmarkWebgui(w, req, BitmarkWebguiConfigFile, GlobalConfig, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleBitcoind(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-bitcoind")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	switch req.Method {
	case `POST`:
		api.Bitcoind(w, req, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleBitmarkd(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-bitmarkd")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	switch req.Method {
	case `POST`:
		api.Bitmarkd(w, req, BitmarkWebguiConfigFile, GlobalConfig, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleProoferd(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-prooferd")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	switch req.Method {
	case `POST`:
		api.Prooferd(w, req, BitmarkWebguiConfigFile, GlobalConfig, log)
	case `OPTIONS`:
		return
	default:
		log.Error("Error: Unknow method")
	}
}

func handleBitmarkConsole(w http.ResponseWriter, req *http.Request) {
	log := logger.New("api-bitmarkConsoleProxy")
	api.SetCORSHeader(w, req)

	if req.Method == "OPTIONS" || !checkAuthorization(w, req, true, log) {
		return
	}

	req.URL.Path = req.URL.Path[8:]
	retry := 3
	for i := 0; i < retry; i++ {
		if !bitmarkConsoleService.IsRunning() {
			if err := bitmarkConsoleService.StartBitmarkConsole(); nil != err {
				log.Errorf("start bitmarkConsole server fail: %v\n", err)

			} else {
				break
			}
		} else {
			break
		}
		time.Sleep(time.Second * 2)
	}

	bitmarkConsoleProxy.ServeHTTP(w, req)
}

func handleBitmarkConsoleWS() websocket.Handler {
	log := logger.New("api-bitmarkConsoleWS")

	return websocket.Handler(func(w *websocket.Conn) {

		origin := "http://localhost:" + bitmarkConsoleService.Port() + "/"
		url := "ws://localhost:" + bitmarkConsoleService.Port() + "/ws"
		ws, err := websocket.Dial(url, "", origin)
		if err != nil {
			log.Errorf("websocket dial fail: %v\n", err)
			return
		}
		defer func() {
			ws.Close()
			bitmarkConsoleService.StopBitmarkConsole()
		}()

		deadLineReader := &DeadlineReader{
			w: w,
		}
		go io.Copy(w, ws)
		io.Copy(ws, deadLineReader)
	})
}

type DeadlineReader struct {
	w *websocket.Conn
}

func (reader *DeadlineReader) Read(p []byte) (n int, err error) {
	reader.w.SetDeadline(time.Now().Add(5 * time.Minute))
	return reader.w.Read(p)
}
