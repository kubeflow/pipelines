// Copyright 2020 Google LLC
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
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/cache/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	TLSDir      string = "/etc/webhook/certs"
	TLSCertFile string = "cert.pem"
	TLSKeyFile  string = "key.pem"
)

const (
	MutateAPI   string = "/mutate"
	WebhookPort string = ":8443"
)

const (
	initConnectionTimeout = "InitConnectionTimeout"

	mysqlDBDriverDefault            = "mysql"
	mysqlDBHostDefault              = "mysql"
	mysqlDBPortDefault              = "3306"
	mysqlDBGroupConcatMaxLenDefault = "4194304"
)

type WhSvrDBParameters struct {
	dbDriver            string
	dbHost              string
	dbPort              string
	dbUser              string
	dbPwd               string
	dbGroupConcatMaxLen string
}

func main() {
	var params WhSvrDBParameters
	flag.StringVar(&params.dbDriver, "db_driver", mysqlDBDriverDefault, "Database driver name, mysql is the default value")
	flag.StringVar(&params.dbHost, "db_host", mysqlDBHostDefault, "Database host name.")
	flag.StringVar(&params.dbPort, "db_port", mysqlDBPortDefault, "Database port number.")
	flag.StringVar(&params.dbUser, "db_user", "root", "Database user name.")
	flag.StringVar(&params.dbPwd, "db_password", "", "Database password.")
	flag.StringVar(&params.dbGroupConcatMaxLen, "db_group_concat_max_len", mysqlDBGroupConcatMaxLenDefault, "Database password.")

	log.Println("Initing client manager....")
	clientManager := NewClientManager(params)

	go WatchPods()

	certPath := filepath.Join(TLSDir, TLSCertFile)
	keyPath := filepath.Join(TLSDir, TLSKeyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateAPI, server.AdmitFuncHandler(server.MutatePodIfCached, &clientManager))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    WebhookPort,
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}

func WatchPods() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf(err.Error())
	}
	for {
		listOptions := metav1.ListOptions{
			Watch:         true,
			LabelSelector: "workflows.argoproj.io/workflow",
		}
		// pods, err := clientset.CoreV1().Pods("kubeflow").List(listOptions)
		watcher, err := clientset.CoreV1().Pods("kubeflow").Watch(listOptions)

		if err != nil {
			log.Printf("watcher error:" + err.Error())
		}
		// for _, pod := range pods.Items {
		// 	log.Printf(pod.ObjectMeta.Name)
		// }
		for event := range watcher.ResultChan() {
			// pod := event.Object.(*corev1.Pod)
			pod := reflect.ValueOf(event.Object).Interface().(*corev1.Pod)
			log.Printf((*pod).GetName())
		}
		// pod := reflect.ValueOf(events.Object).Interface().(*corev1.Pod)
		// // pod := events.Object.(*corev1.Pod)
		// log.Printf((*pod).Kind)
		// log.Printf((*pod).ObjectMeta.Name)
	}
}
