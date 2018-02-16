package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"io/ioutil"
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
	glog.Infof("get a new default request")
	w.Write([]byte("Nothing is here. \n"))
}

func main() {
	flag.Parse()

	glog.Infof("starting web server")

	var config Config
	if *configPath != "" {
		b, err := ioutil.ReadFile(*configPath)
		if err != nil {
			glog.Fatalf("Failed to read config at %s: %v", *configPath, err)
		}
		if err := json.Unmarshal(b, &config); err != nil {
			glog.Fatalf("Failed to parse config file at %s: %v", *configPath, err)
		}
	}

	router := mux.NewRouter()

	clientManager := NewClientManager(config)
	restAPIHandler := CreateRestAPIHandler(clientManager)
	router.PathPrefix(apiRouter).Handler(http.StripPrefix(apiRouter, restAPIHandler))

	// TODO: Better exception handling
	router.HandleFunc("/", DefaultHandler)

	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *portFlag), router))

	clientManager.End()
}
