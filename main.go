package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	configFlag     = flag.String("cfg", "config.json", "path to config file")
	listenAddrFlag = flag.String("http", "127.0.0.1:5000", "forwarding http listen addr")
)

var (
	configMu sync.RWMutex
	config   = make(map[string]string)
)

func main() {
	flag.Parse()

	if err := loadConfig(); err != nil {
		os.Exit(1)
	}

	forwardClient := http.Client{Timeout: 5 * time.Second}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		configMu.RLock()
		target, ok := config[r.URL.Path]
		configMu.RUnlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		log.Printf("got request %q -> %q", r.URL.Path, target)
		resp, err := forwardClient.Get(target)
		if err != nil {
			log.Printf("forward: request failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("forward: failed to copy response: %v", err)
		}
	})

	go handleSignals()
	log.Printf("http: starting server at %q...", *listenAddrFlag)
	if err := http.ListenAndServe(*listenAddrFlag, nil); err != nil {
		log.Printf("http listener on %q failed: %v", *listenAddrFlag, err)
		os.Exit(2)
	}
}

func loadConfig() error {
	log.Printf("loading config from %q", *configFlag)

	bs, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Printf("failed to open config file at %q: %v", *configFlag, err)
		return err
	}

	cfg := make(map[string]string)
	if err := json.Unmarshal(bs, &cfg); err != nil {
		log.Printf("failed to load config file content: %v", err)
		return err
	}

	log.Printf("loaded %d entities from config file", len(cfg))

	configMu.Lock()
	defer configMu.Unlock()
	config = cfg

	return nil
}

func handleSignals() {
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGUSR1, syscall.SIGUSR2)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM)

	for {
		select {
		case sig := <-reload:
			log.Printf("got %q signal, reloading config...", sig.String())
			_ = loadConfig()
		case sig := <-terminate:
			log.Printf("got %q signal, exiting...", sig)
			os.Exit(0)
		}
	}
}
