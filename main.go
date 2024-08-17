package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	configFlag     = flag.String("cfg", "config.json", "path to config file")
	listenAddrFlag = flag.String("http", "127.0.0.1:5000", "forwarding http listen addr")
)

func main() {
	flag.Parse()

	log.Printf("loading config from %q", *configFlag)

	fd, err := os.Open(*configFlag)
	if err != nil {
		log.Printf("failed to open config file at %q: %v", *configFlag, err)
		os.Exit(1)
	}

	config := make(map[string]string)
	if err := json.NewDecoder(fd).Decode(&config); err != nil {
		log.Printf("failed to load config file content: %v", err)
		os.Exit(1)
	}

	log.Printf("loaded %d entities from config file", len(config))

	forwardClient := http.Client{Timeout: 5 * time.Second}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		target, ok := config[r.URL.Path]
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

	log.Printf("http: starting server at %q...", *listenAddrFlag)
	if err := http.ListenAndServe(*listenAddrFlag, nil); err != nil {
		log.Printf("http listener on %q failed: %v", *listenAddrFlag, err)
		os.Exit(2)
	}
}
