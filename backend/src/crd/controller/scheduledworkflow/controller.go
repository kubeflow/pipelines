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

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	workflowinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/client"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfScheme "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/scheme"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	wraperror "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	Workflow          = "Workflow"
	ScheduledWorkflow = "ScheduledWorkflow"
)

var (
	// DefaultJobBackOff is the max backoff period
	DefaultJobBackOff = 10 * time.Second
	// MaxJobBackOff is the max backoff period
	MaxJobBackOff = 360 * time.Second
)

// Controller is the controller implementation for ScheduledWorkflow resources
type Controller struct {
	kubeClient     *client.KubeClient
	swfClient      *client.ScheduledWorkflowClient
	workflowClient *client.WorkflowClient

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	// An interface to generate the current time.
	time commonutil.TimeInterface

	// the timezone loation which the scheduled will use
	location *time.Location
}

// NewController returns a new sample controller
func NewController(
	kubeClientSet kubernetes.Interface,
	swfClientSet swfclientset.Interface,
	workflowClientSet workflowclientset.Interface,
	swfInformerFactory swfinformers.SharedInformerFactory,
	workflowInformerFactory workflowinformers.SharedInformerFactory,
	time commonutil.TimeInterface,
	location *time.Location) *Controller {

	// obtain references to shared informers
	swfInformer := swfInformerFactory.Scheduledworkflow().V1beta1().ScheduledWorkflows()
	workflowInformer := workflowInformerFactory.Argoproj().V1alpha1().Workflows()

	// Add controller types to the default Kubernetes Scheme so Events can be
	// logged for controller types.
	swfScheme.AddToScheme(scheme.Scheme)

	// Create event broadcaster
	log.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: util.ControllerAgentName})

	controller := &Controller{
		kubeClient:     client.NewKubeClient(kubeClientSet, recorder),
		swfClient:      client.NewScheduledWorkflowClient(swfClientSet, swfInformer),
		workflowClient: client.NewWorkflowClient(workflowClientSet, workflowInformer),
		workqueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(DefaultJobBackOff, MaxJobBackOff), swfregister.Kind),
		time:     time,
		location: location,
	}

	log.Info("Setting up event handlers")

	// Set up an event handler for when the Scheduled Workflow changes
	controller.swfClient.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueScheduledWorkflow,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueScheduledWorkflow(new)
		},
		DeleteFunc: controller.enqueueScheduledWorkflowForDelete,
	})

	// Set up an event handler for when WorkflowHistory resources change. This
	// handler will lookup the owner of the given WorkflowHistory, and if it is
	// owned by a ScheduledWorkflow, it will enqueue that ScheduledWorkflow for
	// processing. This way, we don't need to implement custom logic for
	// handling WorkflowHistory resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	controller.workflowClient.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleWorkflow,
		UpdateFunc: func(old, new interface{}) {
			newWorkflow := new.(*workflowapi.Workflow)
			oldWorkflow := old.(*workflowapi.Workflow)
			if newWorkflow.ResourceVersion == oldWorkflow.ResourceVersion {
				// Periodic resync will send update events for all known Workflows.
				// Two different versions of the same WorkflowHistory will always have different RVs.
				return
			}
			controller.handleWorkflow(new)
		},
		DeleteFunc: controller.handleWorkflow,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting ScheduledWorkflow controller")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh,
		c.workflowClient.HasSynced(),
		c.swfClient.HasSynced()); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	// Launch multiple workers to process ScheduledWorkflows
	log.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	log.Info("Started workers")

	log.Info("Wait for shut down")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue. It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// enqueueScheduledWorkflow takes a ScheduledWorkflow and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ScheduledWorkflow.
func (c *Controller) enqueueScheduledWorkflow(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(fmt.Errorf("Equeuing object: error: %v: %+v", err, obj))
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) enqueueScheduledWorkflowForDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		c.workqueue.Add(key)
	}
}

// handleWorkflow will take any resource implementing metav1.Object and attempt
// to find the ScheduledWorkflow that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that ScheduledWorkflow to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleWorkflow(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Error decoding object, invalid type."))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("Error decoding object tombstone, invalid type."))
			return
		}
		log.WithFields(log.Fields{
			Workflow: object.GetName(),
		}).Infof("Recovered deleted object '%s' from tombstone.", object.GetName())
	}

	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a ScheduledWorkflow, we should not do anything more
		// with it.
		if ownerRef.Kind != swfregister.Kind {
			log.WithFields(log.Fields{
				Workflow: object.GetName(),
			}).Infof("Processing object (%s): owner is not a Scheduled Workflow.", object.GetName())
			return
		}

		swf, err := c.swfClient.Get(object.GetNamespace(), ownerRef.Name)
		if err != nil {
			log.WithFields(log.Fields{
				Workflow: object.GetName(),
			}).Infof("Processing object (%s): ignoring orphaned object of scheduled Workflow (%s).",
				object.GetName(), ownerRef.Name)
			return
		}

		log.WithFields(log.Fields{
			Workflow:          object.GetName(),
			ScheduledWorkflow: ownerRef.Name,
		}).Infof("Processing object (%s): owner is a ScheduledWorkflow (%s).", object.GetName(),
			ownerRef.Name)
		c.enqueueScheduledWorkflow(swf.Get())
		return
	}
	log.WithFields(log.Fields{
		Workflow: object.GetName(),
	}).Infof("Processing object (%s): object has no owner.", object.GetName())
	return
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	return func(obj interface{}) bool {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("Expected string in workqueue but got %#v", obj))
			return true
		}

		// Notes on workqueues:
		// - when using: workqueue.Forget
		//   The item is reprocessed after the next SharedInformerFactory defaultResync.
		// - when using: workqueue.Forget && workqueue.Add()
		//   The item is reprocessed immediately.
		//   This is not recommended as the status changes may not have propagated, leading to
		//   a (recoverable) versioning error.
		// - when using: workqueue.Forget && workqueue.AddAfter(X seconds)
		//   The item is reprocessed after X seconds.
		//   It can be re-processes earlier depending on SharedInformerFactory defaultResync.
		//   Deleting and recreating the resource using kubectl does not trigger early processing.
		// - when using: workqueue.Forget && workqueue.AddRateLimited()
		//   The item is reprocessed after the baseDelay
		// - when using: workqueue.AddRateLimited()
		//   The item is reprocessed folowing the exponential backoff strategy:
		//   baseDelay * 10^(failure count)
		//   It is not reprocessed earlier due to SharedInformerFactory defaultResync.
		//   It is not reprocessed earlier even if the resource is deleted/re-created.
		// - when using: workqueue.Add()
		//   The item is reprocessed immediately (not recommended)
		// - when using: workqueue.AddAfter(X seconds)
		//   The item is reprocessed immediately
		// - when using: nothing
		//   The item is reprocessed using the exponential backoff strategy.

		// Run the syncHandler, passing it the namespace/name string of the
		// ScheduledWorkflow to be synced.
		syncAgain, retryOnError, swf, err := c.syncHandler(key)
		if err != nil && retryOnError {
			// Transient failure. We will retry.
			c.workqueue.AddRateLimited(obj) // Exponential backoff.
			runtime.HandleError(fmt.Errorf("Transient failure: %+v", err))
			if swf != nil {
				c.kubeClient.RecordSyncFailure(swf.Get(),
					fmt.Sprintf("Transient failure: %v", err.Error()))
			}
			return true
		} else if err != nil && !retryOnError {
			// Permanent failure. We won't retry.
			// Will resync after the SharedInformerFactory defaultResync delay.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("Permanent failure: %+v", err))
			if swf != nil {
				c.kubeClient.RecordSyncFailure(swf.Get(),
					fmt.Sprintf("Permanent failure: %v", err.Error()))
			}
			return true
		} else if err == nil && !syncAgain {
			// Success.
			// Will resync after the SharedInformerFactory defaultResync delay.
			c.workqueue.Forget(obj)
			if swf != nil {
				c.kubeClient.RecordSyncSuccess(swf.Get(), "All done")
			}
			return true
		} else {
			// Success and sync again soon.
			c.workqueue.Forget(obj)
			c.workqueue.AddAfter(obj, 10*time.Second) // Need status changes to propagate.
			if swf != nil {
				c.kubeClient.RecordSyncSuccess(swf.Get(), "Partially done, syncing again soon")
			}
			return true
		}
	}(obj)
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the ScheduledWorkflow
// with the current status of the resource.
func (c *Controller) syncHandler(key string) (
	syncAgain bool, retryOnError bool, swf *util.ScheduledWorkflow, err error) {

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// Permanent failure.
		return false, false, nil,
			wraperror.Wrapf(err, "Invalid resource key (%s): %v", key, err)
	}

	// Get the ScheduledWorkflow with this namespace/name
	swf, err = c.swfClient.Get(namespace, name)
	if err != nil {
		// Permanent failure.
		// The ScheduledWorkflow may no longer exist, we stop processing and do not retry.
		return false, false, nil,
			wraperror.Wrapf(err, "ScheduledWorkflow (%s) in work queue no longer exists: %v", key, err)
	}

	// Get the current time
	// NOTE: call time.Now() only once per event so that all the functions have a consistent
	// number for the current time.
	nowEpoch := c.time.Now().Unix()

	// Get active workflows for this ScheduledWorkflow.
	active, err := c.workflowClient.List(swf.Name,
		false, /* active workflow */
		0 /* retrieve all workflows */)
	if err != nil {
		return false, true, swf,
			wraperror.Wrapf(err, "Syncing ScheduledWorkflow (%v): transient failure, can't fetch active workflows: %v", name, err)
	}

	// Get completed workflows for this ScheduledWorkflow.
	completed, err := c.workflowClient.List(swf.Name,
		true, /* completed workflows */
		swf.MinIndex())
	if err != nil {
		return false, true, swf,
			wraperror.Wrapf(err, "Syncing ScheduledWorkflow (%v): transient failure, can't fetch completed workflows: %v", name, err)
	}

	workflow, nextScheduledEpoch, err := c.submitNextWorkflowIfNeeded(swf, len(active), nowEpoch)
	if err != nil {
		return false, true, swf,
			wraperror.Wrapf(err, "Syncing ScheduledWorkflow (%v): transient failure, can't fetch completed workflows: %v", name, err)
	}

	err = c.updateStatus(swf, workflow, active, completed, nextScheduledEpoch, nowEpoch)
	if err != nil {
		return false, true, swf,
			wraperror.Wrapf(err, "Syncing ScheduledWorkflow (%v): transient failure, can't update swf status: %v", name, err)
	}

	if workflow != nil {
		// Success. Since we created a new workflow, sync again soon since there might be one more
		// resource to create.
		log.WithFields(log.Fields{
			ScheduledWorkflow: name,
		}).Infof("Syncing ScheduledWorkflow (%v): success, requeuing for further processing.", name)
		return true, false, swf, nil
	}

	// Success. We did not create any new resource. We can sync again when something changes.
	log.WithFields(log.Fields{
		ScheduledWorkflow: name,
	}).Infof("Syncing ScheduledWorkflow (%v): success, processing complete.", name)
	return false, false, swf, nil
}

// Submits the next workflow if a workflow is due to execute. Returns the submitted workflow,
// an error (if any), and a boolean indicating (in case of an error) whether handling the
// ScheduledWorkflow should be attempted again at a later time.
func (c *Controller) submitNextWorkflowIfNeeded(swf *util.ScheduledWorkflow,
	activeWorkflowCount int, nowEpoch int64) (
	workflow *commonutil.Workflow, nextScheduledEpoch int64, err error) {
	// Compute the next scheduled time.
	nextScheduledEpoch, shouldRunNow := swf.GetNextScheduledEpoch(
		int64(activeWorkflowCount), nowEpoch, *c.location)

	if !shouldRunNow {
		log.WithFields(log.Fields{
			ScheduledWorkflow: swf.Name,
		}).Infof("Submitting workflow for ScheduledWorkflow (%v): nothing to submit (next scheduled at: %v)",
			swf.Name, commonutil.FormatTimeForLogging(nextScheduledEpoch))
		return nil, nextScheduledEpoch, nil
	}

	workflow, err = c.submitNewWorkflowIfNotAlreadySubmitted(swf, nextScheduledEpoch, nowEpoch)
	if err != nil {
		log.WithFields(log.Fields{
			ScheduledWorkflow: swf.Name,
		}).Errorf("Submitting workflow for ScheduledWorkflow (%v): transient error while submitting workflow: %v",
			swf.Name, err)
		// There was an error submitting a new workflow.
		// We should attempt to handle the schedule again at a later time.
		return nil, nextScheduledEpoch, err
	}
	log.WithFields(log.Fields{
		ScheduledWorkflow: swf.Name,
		Workflow:          workflow.Get().Name,
	}).Infof("Submitting workflow for ScheduledWorkflow (%v): workflow (%v) successfully submitted (scheduled at: %v)",
		swf.Name, workflow.Get().Name, commonutil.FormatTimeForLogging(nextScheduledEpoch))
	return workflow, nextScheduledEpoch, nil
}

func (c *Controller) submitNewWorkflowIfNotAlreadySubmitted(
	swf *util.ScheduledWorkflow, nextScheduledEpoch int64, nowEpoch int64) (
	*commonutil.Workflow, error) {

	workflowName := swf.NextResourceName()

	// Try to fetch this workflow
	// If it already exists, it means that it was already created in a previous iteration
	// of this controller but that the controller failed to save this data.
	foundWorkflow, isNotFoundError, err := c.workflowClient.Get(swf.Namespace,
		workflowName)
	if err == nil {
		// The workflow was already created by a previous iteration of this controller.
		// Nothing to do except returning the information needed by the controller to update
		// the ScheduledWorkflow status.
		return foundWorkflow, nil
	}

	if !isNotFoundError {
		// There was an error while attempting to retrieve the workflow
		return nil, err
	}

	// If the workflow is not found, we need to create it.
	newWorkflow, err := swf.NewWorkflow(nextScheduledEpoch, nowEpoch)
	createdWorkflow, err := c.workflowClient.Create(swf.Namespace, newWorkflow)

	if err != nil {
		return nil, err
	}
	return createdWorkflow, nil
}

func (c *Controller) updateStatus(
	swf *util.ScheduledWorkflow,
	workflow *commonutil.Workflow,
	active []swfapi.WorkflowStatus,
	completed []swfapi.WorkflowStatus,
	nextScheduledEpoch int64,
	nowEpoch int64) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	swfCopy := util.NewScheduledWorkflow(swf.Get().DeepCopy())
	swfCopy.UpdateStatus(nowEpoch, workflow, nextScheduledEpoch, active, completed, c.location)

	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the ScheduledWorkflow. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	return c.swfClient.Update(swf.Namespace, swfCopy)
}
