package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

var (
	portFlag   = flag.Int("port", 8888, "Port to listen on")
	configPath = flag.String("config", "", "Path to JSON file containing config")
)

func main() {
	flag.Parse()
	glog.Infof("starting web server")

	clientManager := NewClientManager(*configPath)

	app := newApp(clientManager)
	app.Run(iris.Addr(fmt.Sprintf(":%d", *portFlag)), iris.WithOptimizations)

	clientManager.End()
}
