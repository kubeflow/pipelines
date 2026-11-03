// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	errorutil "github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	// DefaultJobBackOff is the default backoff period
	DefaultJobBackOff = 1 * time.Second
	// MaxJobBackOff is the max backoff period
	MaxJobBackOff = 360 * time.Second
)

type Saver interface {
	Save(key string, namespace string, name string, nowEpoch int64) error
}

type EventHandler interface {
	AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

// PersistenceWorker is a generic worker to persist objects from a queue.
type PersistenceWorker struct {
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time.
	workqueue workqueue.RateLimitingInterface

	// An interface to generate the current time.
	time                 util.TimeInterface
	enforceRequeueDelays bool
	saver                Saver
}

// NewPersistenceWorker returns a new PersistenceWorker
func NewPersistenceWorker(
	time util.TimeInterface,
	name string,
	eventHandler util.ExecutionInformerEventHandler,
	enforceRequeueDelays bool,
	saver Saver,
) (*PersistenceWorker, error) {
	worker := &PersistenceWorker{
		workqueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(DefaultJobBackOff, MaxJobBackOff), name),
		time:                 time,
		enforceRequeueDelays: enforceRequeueDelays,
		saver:                saver,
	}

	log.Info("Setting up event handlers")

	// Set up an event handler for when the Scheduled Workflow changes
	_, err := eventHandler.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: worker.enqueue,
		UpdateFunc: func(old, new interface{}) {
			worker.enqueue(new)
		},
		DeleteFunc: worker.enqueueForDelete,
	})
	if err != nil {
		return nil, err
	}

	return worker, nil
}

func (p *PersistenceWorker) Shutdown() {
	p.workqueue.ShutDown()
}

func (p *PersistenceWorker) Len() int {
	return p.workqueue.Len()
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue. It enforces that the syncHandler is never invoked concurrently with the same key.
func (p *PersistenceWorker) RunWorker() {
	for p.processNextWorkItem() {
	}
}

// enqueue takes a Workflow or a ScheduledWorkflow and converts it
// into a namespace/name string which is then put onto the work queue.
func (p *PersistenceWorker) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(fmt.Errorf("Equeuing object: error: %v: %+v", err, obj))
		return
	}
	// New state is ready to persist immediately. Only failed saves consume retry
	// backoff; ordinary status updates must not inflate the failure count.
	p.workqueue.Add(key)
}

func (p *PersistenceWorker) enqueueForDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		p.workqueue.Add(key)
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (p *PersistenceWorker) processNextWorkItem() bool {
	obj, shutdown := p.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer p.workqueue.Done.
	return func(obj interface{}) bool {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer p.workqueue.Done(obj)
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
			p.workqueue.Forget(obj)
			log.Errorf("Expected string in workqueue but got %#v", obj)
			return true
		}

		// Retries back off until another informer event makes fresh state ready.
		// Successful saves forget the failure count; the queue still coalesces
		// duplicate events and prevents concurrent processing of the same key.

		// Run the syncHandler, passing it the namespace/name string of the
		// resource to be synced.
		err := p.syncHandler(key)
		retryOnError := errorutil.HasCustomCode(err, errorutil.CUSTOM_CODE_TRANSIENT)
		if err != nil && retryOnError {
			// Transient failure. We will retry.
			log.Errorf("Transient failure while syncing resource (%v): %+v", key, err)
			if p.enforceRequeueDelays {
				p.workqueue.AddRateLimited(obj) // Exponential backoff.
			} else {
				p.workqueue.Add(obj) // For testing.
			}
			return true
		} else if err != nil && !retryOnError {
			// Permanent failure. We won't retry.
			// Will resync after the SharedInformerFactory defaultResync delay.
			log.Errorf("Permanent failure while syncing resource (%v): %+v", key, err)
			p.workqueue.Forget(obj)
			return true
		} else {
			// Success.
			// Will resync after the SharedInformerFactory defaultResync delay.
			log.Infof("Success while syncing resource (%v)", key)
			p.workqueue.Forget(obj)
			return true
		}
	}(obj)
}

// syncHandler picks items from the queue and passes them to the saver, which,
// in turn, calls Report[Scheduled]Workflow to sync it with the DB
func (p *PersistenceWorker) syncHandler(key string) error {

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// Permanent failure.
		return errorutil.NewCustomError(err, errorutil.CUSTOM_CODE_PERMANENT,
			"Invalid resource key (%s): %v", key, err)
	}

	// Get the current time
	// NOTE: call time.Now() only once per event so that all the functions have a consistent
	// number for the current time.
	nowEpoch := p.time.Now().Unix()

	return p.saver.Save(key, namespace, name, nowEpoch)
}
