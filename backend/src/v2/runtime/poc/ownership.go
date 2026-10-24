package poc

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// ownershipGuard binds backend operations to one immutable lease incarnation.
type ownershipGuard struct {
	lifetime context.Context
	check    func(context.Context) error
	mu       sync.Mutex
	attempts map[string]struct{}
}

func (g *ownershipGuard) operation(parent context.Context) (context.Context, context.CancelFunc, error) {
	if err := parent.Err(); err != nil {
		return nil, nil, err
	}
	if g == nil {
		return parent, func() {}, nil
	}
	if err := g.lifetime.Err(); err != nil {
		return nil, nil, context.Cause(g.lifetime)
	}
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(g.lifetime, cancel)
	cleanup := func() {
		stop()
		cancel()
	}
	if err := g.check(ctx); err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		cleanup()
		return nil, nil, err
	}
	return ctx, cleanup, nil
}

func (g *ownershipGuard) trackAttempt(attemptID string) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.attempts[attemptID] = struct{}{}
}

func (g *ownershipGuard) releaseAttempts(c *Coordinator) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for attemptID := range g.attempts {
		c.releaseAttempt(attemptID)
	}
}

type ownedExecutor struct {
	WorkloadExecutorPlugin
	guard *ownershipGuard
}

func (e ownedExecutor) Submit(ctx context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return ExecutionReference{}, err
	}
	defer cancel()
	return e.WorkloadExecutorPlugin.Submit(ctx, request)
}

func (e ownedExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	defer cancel()
	return e.WorkloadExecutorPlugin.Wait(ctx, reference)
}

func (e ownedExecutor) Cancel(ctx context.Context, reference ExecutionReference) (CancellationResult, error) {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return CancellationResult{State: CancellationUnknown}, err
	}
	defer cancel()
	return e.WorkloadExecutorPlugin.Cancel(ctx, reference)
}

func (e ownedExecutor) Recover(ctx context.Context, identity TaskAttemptIdentity, reference *ExecutionReference) (RecoveryResult, error) {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return RecoveryResult{State: RecoveryUnknown}, err
	}
	defer cancel()
	recoverer, ok := e.WorkloadExecutorPlugin.(LostTaskRecoverer)
	if !ok {
		return RecoveryResult{State: RecoveryUnknown}, fmt.Errorf("executor %q does not implement lost-task recovery", e.Type())
	}
	return recoverer.Recover(ctx, identity, reference)
}

func (e ownedExecutor) Cleanup(ctx context.Context, reference ExecutionReference) error {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return e.WorkloadExecutorPlugin.Cleanup(ctx, reference)
}

func (e ownedExecutor) Logs(ctx context.Context, reference ExecutionReference) (ExecutionLogSource, error) {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return ExecutionLogSource{}, err
	}
	defer cancel()
	source, err := e.WorkloadExecutorPlugin.Logs(ctx, reference)
	if err != nil {
		return ExecutionLogSource{}, err
	}
	return ExecutionLogSource{Identity: source.Identity, stream: func(ctx context.Context, follow bool, destination io.Writer) error {
		ctx, cancel, err := e.guard.operation(ctx)
		if err != nil {
			return err
		}
		defer cancel()
		return source.Stream(ctx, follow, destination)
	}}, nil
}

func (e ownedExecutor) ReadLogs(ctx context.Context, reference ExecutionReference, follow bool, destination io.Writer) error {
	ctx, cancel, err := e.guard.operation(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return streamExecutionLogs(ctx, e.WorkloadExecutorPlugin, reference, follow, destination)
}

type ownedWorkspaceProvisioner struct {
	WorkspaceProvisionerPlugin
	guard *ownershipGuard
}

func (p ownedWorkspaceProvisioner) Provision(ctx context.Context, request WorkspaceRequest) (*WorkspaceHandle, error) {
	ctx, cancel, err := p.guard.operation(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	return p.WorkspaceProvisionerPlugin.Provision(ctx, request)
}

func (p ownedWorkspaceProvisioner) ResolveAttachment(ctx context.Context, reference *WorkspaceHandle, target ExecutionTarget) (*WorkspaceHandle, error) {
	ctx, cancel, err := p.guard.operation(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	return p.WorkspaceProvisionerPlugin.ResolveAttachment(ctx, reference, target)
}

func (p ownedWorkspaceProvisioner) Cleanup(ctx context.Context, reference *WorkspaceHandle) error {
	ctx, cancel, err := p.guard.operation(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return p.WorkspaceProvisionerPlugin.Cleanup(ctx, reference)
}
