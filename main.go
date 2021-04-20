package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/MiningPool0826/xmrpool/pool"
	"github.com/MiningPool0826/xmrpool/stratum"

	"github.com/goji/httpauth"
	"github.com/gorilla/mux"
)

var (
	LatestTag           = ""
	LatestTagCommitSHA1 = ""
	LatestCommitSHA1    = ""
	BuildTime           = ""
)

var cfg pool.Config

func startStratum() {
	if cfg.Threads > 0 {
		runtime.GOMAXPROCS(cfg.Threads)
		log.Printf("Running with %v threads", cfg.Threads)
	} else {
		n := runtime.NumCPU()
		runtime.GOMAXPROCS(n)
		log.Printf("Running with default %v threads", n)
	}

	s := stratum.NewStratum(&cfg)
	if cfg.Frontend.Enabled {
		go startFrontend(&cfg, s)
	}
	s.Listen()
}

func startFrontend(cfg *pool.Config, s *stratum.StratumServer) {
	r := mux.NewRouter()
	r.HandleFunc("/stats", s.StatsIndex)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))
	var err error
	if len(cfg.Frontend.Password) > 0 {
		auth := httpauth.SimpleBasicAuth(cfg.Frontend.Login, cfg.Frontend.Password)
		err = http.ListenAndServe(cfg.Frontend.Listen, auth(r))
	} else {
		err = http.ListenAndServe(cfg.Frontend.Listen, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

//func startNewrelic() {
//	// Run NewRelic
//	if cfg.NewrelicEnabled {
//		nr := gorelic.NewAgent()
//		nr.Verbose = cfg.NewrelicVerbose
//		nr.NewrelicLicense = cfg.NewrelicKey
//		nr.NewrelicName = cfg.NewrelicName
//		nr.Run()
//	}
//}

func readConfig(cfg *pool.Config) {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatal("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func OptionParse() {
	var showVer bool
	flag.BoolVar(&showVer, "v", false, "show build version")

	flag.Parse()

	if showVer {
		fmt.Printf("Latest Tag: %s\n", LatestTag)
		fmt.Printf("Latest Tag Commit SHA1: %s\n", LatestTagCommitSHA1)
		fmt.Printf("Latest Commit SHA1: %s\n", LatestCommitSHA1)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	OptionParse()
	readConfig(&cfg)
	//startNewrelic()
	startStratum()
}
