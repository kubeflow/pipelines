package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	apiRouter = "/api"
)

var (
	portFlag   = flag.Int("port", 8888, "Port to listen on")
	configPath = flag.String("config", "", "Path to JSON file containing config")
)

type DBConfig struct {
	DriverName     string
	DataSourceName string
}

type Config struct {
	DBConfig DBConfig
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("get a new default request")
	w.Write([]byte("Nothing is here. \n"))
}

func main() {
	flag.Parse()

	log.Printf("starting web server")

	var config Config
	if *configPath != "" {
		b, err := ioutil.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("Failed to read config at %s: %v", *configPath, err)
		}
		if err := json.Unmarshal(b, &config); err != nil {
			log.Fatalf("Failed to parse config file at %s: %v", *configPath, err)
		}
	}

	router := mux.NewRouter()

	cm := NewClientManager(config)
	restAPIHandler := CreateRestAPIHandler(cm)
	router.PathPrefix(apiRouter).Handler(http.StripPrefix(apiRouter, restAPIHandler))

	// TODO: Static contents.
	router.HandleFunc("/", DefaultHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *portFlag), router))

	cm.End()
}
