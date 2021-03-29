package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	kubeflowClient "github.com/kubeflow/pipelines/backend/api/go_client"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	//kubeconfig := filepath.Join(
	//	os.Getenv("HOME"), ".kube", "config",
	//)
	namespace := "kubeflow"
	//k8sClient, err := getClient(kubeconfig)
	k8sClient, err := getClientInside()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	svcs, err := k8sClient.Services(namespace).Get("ml-pipeline", metav1.GetOptions{})
	// panic: services is forbidden: User "system:serviceaccount:kubeflow:default"
	// cannot list resource "services" in API group "" in the namespace "kubeflow"
	// Need to give pods right through service accounts.
	if err != nil {
		panic(err)
	}
	fmt.Println(svcs)

	//https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/
	//hostname := os.Getenv("ML_PIPELINE_SERVICE_HOST")

	hostname := svcs.Spec.ClusterIP
	if len(hostname) <= 0 {
		hostname = "0.0.0.0"
	}

	conn, err := grpc.Dial(hostname+":"+"8887", grpc.WithInsecure())

	if err != nil {
		panic(err)
	}
	client := kubeflowClient.NewRunServiceClient(conn)
	run := kubeflowClient.Run{
		PipelineSpec: &kubeflowClient.PipelineSpec{
			PipelineId: "4c9ae80f-4ae3-404a-9dfa-3fd707af9b93",
		},
		Name: "RemoteRunNiklas",
		//ResourceReferences: []*kubeflowClient.ResourceReference{&kubeflowClient.ResourceReference{}},
	}
	request := kubeflowClient.CreateRunRequest{Run: &run}
	runDetail, err := client.CreateRun(context.Background(), &request)
	if err != nil {
		fmt.Println("HERERE ")
		fmt.Println("HERERE ")
		fmt.Println("HERERE ")
		panic(err)
	}
	fmt.Println(runDetail)
}

func getClientInside() (typev1.CoreV1Interface, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	return clientset.CoreV1(), nil
}

// outside of cluster with kubeconfig
func getClient(configLocation string) (typev1.CoreV1Interface, error) {
	kubeconfig := filepath.Clean(configLocation)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1(), nil
}
