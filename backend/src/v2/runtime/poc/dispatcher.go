package poc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

type dispatchTicket struct {
	retry   bool
	granted bool
	ready   chan struct{}
}

// taskDispatcher accounts for active workloads across all runs of an executor.
type taskDispatcher struct {
	mu          sync.Mutex
	limit       int
	active      int
	retryStreak int
	waiting     []*dispatchTicket
}

func validateExecutor(executor WorkloadExecutorPlugin, multiUser bool) error {
	if limit := executor.Capacity().MaxActiveTasks; limit != nil && *limit <= 0 {
		return fmt.Errorf("executor %q capacity must be positive or unset", executor.Type())
	}
	if executor.Capabilities().RecoverLostTasks {
		if _, ok := executor.(LostTaskRecoverer); !ok {
			return fmt.Errorf("executor %q advertises recovery without implementing LostTaskRecoverer", executor.Type())
		}
		if multiUser && !executor.Capabilities().SupportsMultiUser {
			return fmt.Errorf("executor %q does not support multi-user deployments; configure a single-user installation", executor.Type())
		}
	}
	return nil
}

func (c *Coordinator) dispatcher(executor WorkloadExecutorPlugin) *taskDispatcher {
	c.dispatchMu.Lock()
	defer c.dispatchMu.Unlock()
	dispatcher := c.dispatchers[executor.Type()]
	if dispatcher == nil {
		dispatcher = &taskDispatcher{}
		if limit := executor.Capacity().MaxActiveTasks; limit != nil {
			dispatcher.limit = *limit
		}
		c.dispatchers[executor.Type()] = dispatcher
	}
	return dispatcher
}

func (c *Coordinator) reserveRecoveredAttempt(executor WorkloadExecutorPlugin, attemptID string) {
	c.reservationMu.Lock()
	defer c.reservationMu.Unlock()
	if _, exists := c.reservations[attemptID]; exists {
		return
	}
	dispatcher := c.dispatcher(executor)
	dispatcher.mu.Lock()
	dispatcher.active++
	dispatcher.mu.Unlock()
	c.reservations[attemptID] = c.trackedAttemptRelease(attemptID, dispatcher.release)
}

func (c *Coordinator) acquireAttempt(ctx context.Context, executor WorkloadExecutorPlugin, attemptID string, retry bool) (func(), error) {
	c.reservationMu.Lock()
	if release, exists := c.reservations[attemptID]; exists {
		c.reservationMu.Unlock()
		return release, nil
	}
	c.reservationMu.Unlock()
	release, err := c.dispatcher(executor).acquire(ctx, retry)
	if err != nil {
		return nil, err
	}
	c.reservationMu.Lock()
	if existing, ok := c.reservations[attemptID]; ok {
		c.reservationMu.Unlock()
		release()
		return existing, nil
	}
	tracked := c.trackedAttemptRelease(attemptID, release)
	c.reservations[attemptID] = tracked
	c.reservationMu.Unlock()
	return tracked, nil
}

func (c *Coordinator) trackedAttemptRelease(attemptID string, release func()) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			c.reservationMu.Lock()
			delete(c.reservations, attemptID)
			c.reservationMu.Unlock()
			release()
		})
	}
}

func (c *Coordinator) releaseAttempt(attemptID string) {
	c.reservationMu.Lock()
	release := c.reservations[attemptID]
	c.reservationMu.Unlock()
	if release != nil {
		release()
	}
}

func (d *taskDispatcher) acquire(ctx context.Context, retry bool) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ticket := &dispatchTicket{retry: retry, ready: make(chan struct{})}
	d.mu.Lock()
	d.waiting = append(d.waiting, ticket)
	d.dispatch()
	d.mu.Unlock()
	select {
	case <-ticket.ready:
		var once sync.Once
		return func() { once.Do(d.release) }, nil
	case <-ctx.Done():
		d.mu.Lock()
		if ticket.granted {
			d.active--
		} else {
			for index, pending := range d.waiting {
				if pending == ticket {
					d.waiting = append(d.waiting[:index], d.waiting[index+1:]...)
					break
				}
			}
		}
		d.dispatch()
		d.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (d *taskDispatcher) release() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.active--
	d.dispatch()
}

func (d *taskDispatcher) dispatch() {
	for len(d.waiting) > 0 && (d.limit == 0 || d.active < d.limit) {
		index := 0
		preferRetry := d.retryStreak < 3
		for candidate, ticket := range d.waiting {
			if ticket.retry == preferRetry {
				index = candidate
				break
			}
		}
		ticket := d.waiting[index]
		d.waiting = append(d.waiting[:index], d.waiting[index+1:]...)
		ticket.granted = true
		d.active++
		if ticket.retry {
			d.retryStreak++
		} else {
			d.retryStreak = 0
		}
		close(ticket.ready)
	}
}

func toJSONData(value any) (model.JSONData, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result model.JSONData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func fromJSONData(value map[string]interface{}, target any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
