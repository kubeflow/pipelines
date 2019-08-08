// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

func initConfig() {
	// Import environment variable, support nested vars e.g. OBJECTSTORECONFIG_ACCESSKEY
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
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

func getStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}

func getStringConfigWithDefault(configName, value string) string {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetString(configName)
}

func getBoolConfigWithDefault(configName string, value bool) bool {
	if !viper.IsSet(configName) {
		return value
	}
	value, err := strconv.ParseBool(viper.GetString(configName))
	if err != nil {
		glog.Fatalf("Failed converting string to bool %s", viper.GetString(configName))
	}
	return value
}

func getDurationConfig(configName string) time.Duration {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetDuration(configName)
}
