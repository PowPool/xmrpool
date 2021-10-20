package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/MiningPool0826/xmrpool/cnutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/MiningPool0826/xmrpool/payouts"
	"github.com/MiningPool0826/xmrpool/pool"
	"github.com/MiningPool0826/xmrpool/storage"
	"github.com/MiningPool0826/xmrpool/stratum"
	. "github.com/MiningPool0826/xmrpool/util"
)

var (
	LatestTag           = ""
	LatestTagCommitSHA1 = ""
	LatestCommitSHA1    = ""
	BuildTime           = ""
)

var cfg pool.Config
var backend *storage.RedisClient = nil

func startStratum() {
	if cfg.Threads > 0 {
		runtime.GOMAXPROCS(cfg.Threads)
		Info.Printf("Running with %v threads", cfg.Threads)
	} else {
		n := runtime.NumCPU()
		runtime.GOMAXPROCS(n)
		Info.Printf("Running with default %v threads", n)
	}

	s := stratum.NewStratum(&cfg, backend)
	if cfg.Frontend.Enabled {
		go startFrontend(&cfg, s)
	}

	if cfg.Tls.Enabled {
		s.ListenTLS()
	} else {
		s.Listen()
	}
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
		Error.Fatal(err)
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

func readSecurityPass() ([]byte, error) {
	fmt.Printf("Enter Security Password: ")
	var fd int
	if terminal.IsTerminal(int(syscall.Stdin)) {
		fd = int(syscall.Stdin)
	} else {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return nil, errors.New("error allocating terminal")
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}

	SecurityPass, err := terminal.ReadPassword(fd)
	if err != nil {
		return nil, err
	}
	return SecurityPass, nil
}

func decryptPoolConfigure(cfg *pool.Config, passBytes []byte) error {
	b, err := Ae64Decode(cfg.AddressEncrypted, passBytes)
	if err != nil {
		return err
	}
	cfg.Address = string(b)

	// check address
	if !cnutil.ValidateAddress(cfg.Address) {
		return errors.New("decryptPoolConfigure: ValidateAddress")
	}

	if cfg.Redis.Enabled {
		b, err = Ae64Decode(cfg.Redis.PasswordEncrypted, passBytes)
		if err != nil {
			return err
		}
		cfg.Redis.Password = string(b)
	}

	if cfg.RedisFailover.Enabled {
		b, err = Ae64Decode(cfg.RedisFailover.PasswordEncrypted, passBytes)
		if err != nil {
			return err
		}
		cfg.RedisFailover.Password = string(b)
	}

	return nil
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

func startBlockUnlocker() {
	u := payouts.NewBlockUnlocker(&cfg.BlockUnlocker, backend)
	u.Start()
}

func main() {
	OptionParse()
	readConfig(&cfg)
	rand.Seed(time.Now().UTC().UnixNano())

	// init log file
	_ = os.Mkdir("logs", os.ModePerm)
	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	InitLog(iLogFile, eLogFile, sLogFile, bLogFile, cfg.Log.LogSetLevel)

	// set rlimit nofile value
	SetRLimit(100000)

	secPassBytes, err := readSecurityPass()
	if err != nil {
		Error.Fatal("Read Security Password error: ", err.Error())
	}

	err = decryptPoolConfigure(&cfg, secPassBytes)
	if err != nil {
		Error.Fatal("Decrypt Pool Configure error: ", err.Error())
	}

	if cfg.Redis.Enabled {
		backend = storage.NewRedisClient(&cfg.Redis, cfg.Coin)
	} else if cfg.RedisFailover.Enabled {
		backend = storage.NewRedisFailoverClient(&cfg.RedisFailover, cfg.Coin)
	}

	if backend == nil {
		Error.Fatal("Backend is Nil: maybe redis/redisFailover config is invalid")
	}

	pong, err := backend.Check()
	if err != nil {
		Error.Printf("Can't establish connection to backend: %v", err)
	} else {
		Info.Printf("Backend check reply: %v", pong)
	}

	defer func() {
		if r := recover(); r != nil {
			Error.Println(string(debug.Stack()))
		}
	}()

	if cfg.BlockUnlocker.Enabled {
		go startBlockUnlocker()
	}

	//startNewrelic()
	startStratum()
}
