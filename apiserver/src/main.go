package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

const (
	apiRouterPrefix = "/apis/v1alpha1"
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

func main() {
	flag.Parse()
	glog.Infof("starting web server")

	config := getConfig()
	clientManager := NewClientManager(config)

	app := newApp(clientManager)
	app.Run(iris.Addr(fmt.Sprintf(":%d", *portFlag)), iris.WithOptimizations)

	clientManager.End()
}

func getConfig() Config {
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
	return config
}
