package main

import (
	"context"
	"fmt"
	"os"

	kubeflowClient "github.com/kubeflow/pipelines/backend/api/go_client"
	"google.golang.org/grpc"
)

func main() {

	hostname := os.Getenv("SVC_HOST_NAME")

	if len(hostname) <= 0 {
		hostname = "0.0.0.0"
	}

	conn, err := grpc.Dial("127.0.0.1"+":"+"8887", grpc.WithInsecure())

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
