package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

func initConfig() {
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
}

func getConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}
