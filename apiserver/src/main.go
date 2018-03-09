package main

import (
	"flag"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/kataras/iris"
	"github.com/spf13/viper"
)

var (
	portFlag   = flag.Int("port", 8888, "Port to listen on")
	configPath = flag.String("config", "", "Path to JSON file containing config")
)

func main() {
	flag.Parse()
	glog.Infof("starting web server")

	// Import environment variable
	viper.AutomaticEnv()

	// Set configuration file name. The format is auto detected in this case.
	viper.SetConfigName("config")
	viper.AddConfigPath(*configPath)
	err := viper.ReadInConfig()
	if err != nil {
		glog.Fatalf("Fatal error config file: %s", err)
	}

	// Watch for configuration change
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Read in config again
		viper.ReadInConfig()
	})

	clientManager := NewClientManager()

	app := newApp(clientManager)

	app.Run(
		iris.Addr(fmt.Sprintf(":%d", *portFlag)),
		iris.WithOptimizations,
		// Post limit at 32MB.
		iris.WithPostMaxMemory(32<<20))

	clientManager.End()
}
