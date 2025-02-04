// Copyright 2018 The Kubeflow Authors
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
	"context"
	"fmt"
	"time"

	swfScheme "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/scheme"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"

	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/worker"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PersistenceAgent is an agent to persist resources to a database.
type PersistenceAgent struct {
	k8sclient.Client
	Scheme   *runtime.Scheme
	logger   *log.Logger
	swfsaver *worker.ScheduledWorkflowSaver
	wfsaver  *worker.WorkflowSaver
}

// NewPersistenceAgent creates a new controller-runtime Manager, sets up the scheme, initializes saver instances,
// creates a reconciler with the persistence logic, and registers the controller with the manager.
// It returns the configured manager.
func NewPersistenceAgent(
	swfInformerFactory swfinformers.SharedInformerFactory,
	execInformer util.ExecutionInformer,
	pipelineClient *client.PipelineClient,
	time util.TimeInterface,
) (ctrl.Manager, error) {
	// obtain references to shared informers
	swfInformer := swfInformerFactory.Scheduledworkflow().V1beta1().ScheduledWorkflows()

	scheme := scheme.Scheme
	swfScheme.AddToScheme(scheme)

	swfClient := client.NewScheduledWorkflowClient(swfInformer)
	workflowClient := client.NewWorkflowClient(execInformer)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	// Create saver instances.
	workflowSaver := worker.NewWorkflowSaver(workflowClient, pipelineClient, ttlSecondsAfterWorkflowFinish)
	scheduledWorkflowSaver := worker.NewScheduledWorkflowSaver(swfClient, pipelineClient)

	// Initialize the reconciler with saver logic.
	reconciler := &PersistenceAgent{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		wfsaver:  workflowSaver,
		swfsaver: scheduledWorkflowSaver,
		logger:   log.New(),
	}

	// Set up the controller to watch both ScheduledWorkflow and Workflow resources.
	if err = ctrl.NewControllerManagedBy(mgr).
		For(&util.ScheduledWorkflow{}).
		Watches(&Workflow{},&handler.EnqueueRequestForObject{}).
		Complete(reconciler); err != nil {
		return nil, err
	}

	return mgr, nil
}

// NewPersistenceAgent returns a new persistence agent.
func (r *PersistenceAgent) Reconcile(
	ctx context.Context,
	req reconcile.Request,
) (reconcile.Result, error) {
	nowEpoch := time.Now().Unix()
	// Attempt to fetch a ScheduledWorkflow first.
	swf := &util.ScheduledWorkflow{}
	err := r.Get(ctx, req.NamespacedName, swf)
	if err == nil {
		r.logger.Info("Reconciling ScheduledWorkflow", "namespace", swf.Namespace, "name", swf.Name)
		key := fmt.Sprintf("%s/%s", swf.Namespace, swf.Name)
		if err := r.swfsaver.Save(key, swf.Namespace, swf.Name, nowEpoch); err != nil {
			if util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT) {
				r.logger.Error(err, "Transient error saving ScheduledWorkflow; will retry", "key", key)
				return reconcile.Result{RequeueAfter: 1 * time.Second}, err
			}
			// For permanent errors, log and do not requeue.
			r.logger.Error(err, "Permanent error saving ScheduledWorkflow", "key", key)
			return reconcile.Result{}, nil
		}
		// Successfully saved ScheduledWorkflow; nothing more to do.
		return reconcile.Result{}, nil
	}

	// If not found, try to fetch a Workflow.
	wf := &Workflow{}
	err = r.Get(ctx, req.NamespacedName, wf)
	if err == nil {
	    r.logger.Info("Reconciling Workflow", "namespace", wf.Namespace, "name", wf.Name)
		key := fmt.Sprintf("%s/%s", wf.Namespace, wf.Name)
		if err := r.wfsaver.Save(key, wf.Namespace, wf.Name, nowEpoch); err != nil {
			if util.HasCustomCode(err, util.CUSTOM_CODE_TRANSIENT) {
				r.logger.Error(err, "Transient error saving Workflow; will retry", "key", key)
				return reconcile.Result{RequeueAfter: 1 * time.Second}, err
			}
			r.logger.Error(err, "Permanent error saving Workflow", "key", key)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, nil
	}

	r.logger.Info("Object not found; may have been deleted", "key", req.NamespacedName)
	return reconcile.Result{}, nil
}

type Workflow struct {
	util.Workflow
}

// Override SetAnnotations on our Workflow type.
// This method will be used instead of util.Workflow's version.
func (w *Workflow) SetAnnotations(val map[string]string) {
	if w.Annotations == nil {
		w.Annotations = make(map[string]string)
	}

	for key, value := range val {
		w.Annotations[key] = value
	}
}

// Override SetLabels on our Workflow type.
// This method will be used instead of util.Workflow's version.
func (w *Workflow) SetLabels(val map[string]string) {
	if w.Labels == nil {
		w.Labels = make(map[string]string)
	}
	for key, value := range val {
		w.Labels[key] = value
	}
}

func (w *Workflow) SetOwnerReferences(refs []metav1.OwnerReference) {
	w.ObjectMeta.OwnerReferences = refs
}