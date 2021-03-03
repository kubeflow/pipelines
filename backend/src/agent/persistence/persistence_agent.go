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
	"fmt"
	"time"

	workflowregister "github.com/argoproj/argo/pkg/apis/workflow"
	workflowinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/worker"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfScheme "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/scheme"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

// PersistenceAgent is an agent to persist resources to a database.
type PersistenceAgent struct {
	swfClient      *client.ScheduledWorkflowClient
	workflowClient *client.WorkflowClient
	swfWorker      *worker.PersistenceWorker
	workflowWorker *worker.PersistenceWorker
}

// NewPersistenceAgent returns a new persistence agent.
func NewPersistenceAgent(
	swfInformerFactory swfinformers.SharedInformerFactory,
	workflowInformerFactory workflowinformers.SharedInformerFactory,
	pipelineClient *client.PipelineClient,
	time util.TimeInterface) *PersistenceAgent {
	// obtain references to shared informers
	swfInformer := swfInformerFactory.Scheduledworkflow().V1beta1().ScheduledWorkflows()
	workflowInformer := workflowInformerFactory.Argoproj().V1alpha1().Workflows()

	// Add controller types to the default Kubernetes Scheme so Events can be
	// logged for controller types.
	swfScheme.AddToScheme(scheme.Scheme)

	swfClient := client.NewScheduledWorkflowClient(swfInformer)
	workflowClient := client.NewWorkflowClient(workflowInformer)

	swfWorker := worker.NewPersistenceWorker(time, swfregister.Kind, swfInformer.Informer(), true,
		worker.NewScheduledWorkflowSaver(swfClient, pipelineClient))

	workflowWorker := worker.NewPersistenceWorker(time, workflowregister.WorkflowKind,
		workflowInformer.Informer(), true,
		worker.NewWorkflowSaver(workflowClient, pipelineClient, ttlSecondsAfterWorkflowFinish))

	agent := &PersistenceAgent{
		swfClient:      swfClient,
		workflowClient: workflowClient,
		swfWorker:      swfWorker,
		workflowWorker: workflowWorker,
	}

	log.Info("Setting up event handlers")

	return agent
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (p *PersistenceAgent) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer p.swfWorker.Shutdown()
	defer p.workflowWorker.Shutdown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting The persistence agent")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		p.workflowClient.HasSynced(),
		p.swfClient.HasSynced()); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	// Launch multiple workers to process ScheduledWorkflows
	log.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(p.swfWorker.RunWorker, time.Second, stopCh)
		go wait.Until(p.workflowWorker.RunWorker, time.Second, stopCh)
	}
	log.Info("Started workers")

	log.Info("Wait for shut down")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}
