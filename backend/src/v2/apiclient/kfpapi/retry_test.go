package kfpapi

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testRetryPolicy() RetryPolicy {
	return RetryPolicy{
		AttemptTimeout: 100 * time.Millisecond,
		TotalTimeout:   300 * time.Millisecond,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	}
}

func TestRetryReadSucceedsImmediately(t *testing.T) {
	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "ok", nil
		})

	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.Equal(t, 1, calls)
}

func TestRetryReadRetriesUnavailable(t *testing.T) {
	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			if calls < 3 {
				return "", status.Error(codes.Unavailable, "temporary")
			}
			return "recovered", nil
		})

	require.NoError(t, err)
	require.Equal(t, "recovered", got)
	require.Equal(t, 3, calls)
}

func TestRetryReadDoesNotRetryPermanentError(t *testing.T) {
	calls := 0
	_, err := retryRead(context.Background(), "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.PermissionDenied, "denied")
		})

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Equal(t, 1, calls)
}

func TestRetryReadRespectsCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	_, err := retryRead(ctx, "TestRetryRead", testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", errors.New("must not run")
		})

	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, calls)
}

func TestRetryReadBoundsHangingAttempt(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 30 * time.Millisecond
	policy.TotalTimeout = 80 * time.Millisecond
	policy.InitialBackoff = 10 * time.Millisecond
	policy.MaxBackoff = 10 * time.Millisecond

	start := time.Now()
	calls := 0
	_, err := retryRead(context.Background(), "TestRetryRead", policy,
		func(ctx context.Context) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		})

	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second)
	require.GreaterOrEqual(t, calls, 1)
	require.LessOrEqual(t, calls, policy.MaxAttempts)
}

func TestRetryReadRetriesAttemptTimeout(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 10 * time.Millisecond
	policy.TotalTimeout = 200 * time.Millisecond

	calls := 0
	got, err := retryRead(context.Background(), "TestRetryRead", policy,
		func(ctx context.Context) (string, error) {
			calls++
			if calls == 1 {
				<-ctx.Done()
				return "", ctx.Err()
			}
			return "recovered", nil
		})

	require.NoError(t, err)
	require.Equal(t, "recovered", got)
	require.Equal(t, 2, calls)
}

func TestRetryReadRespectsCancellationDuringBackoff(t *testing.T) {
	policy := testRetryPolicy()
	policy.InitialBackoff = 200 * time.Millisecond
	policy.MaxBackoff = 200 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attemptStarted := make(chan struct{}, 1)
	start := time.Now()

	done := make(chan error, 1)
	go func() {
		_, err := retryRead(ctx, "TestRetryRead", policy, func(context.Context) (string, error) {
			attemptStarted <- struct{}{}
			return "", status.Error(codes.Unavailable, "temporary")
		})
		done <- err
	}()

	select {
	case <-attemptStarted:
	case <-time.After(time.Second):
		t.Fatal("retry attempt did not start")
	}

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
		require.Less(t, time.Since(start), 150*time.Millisecond)
	case <-time.After(time.Second):
		t.Fatal("retry did not stop promptly after cancellation")
	}
}

type hangingRunServer struct {
	gc.UnimplementedRunServiceServer
}

func (hangingRunServer) GetRun(ctx context.Context, _ *gc.GetRunRequest) (*gc.Run, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestClientAdapterGetRunBoundsHangingGRPC(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	server := grpc.NewServer()
	gc.RegisterRunServiceServer(server, hangingRunServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)

	client := &apiclient.Client{
		Run:  gc.NewRunServiceClient(conn),
		Conn: conn,
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	policy := RetryPolicy{
		AttemptTimeout: 50 * time.Millisecond,
		TotalTimeout:   80 * time.Millisecond,
		MaxAttempts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}

	api := NewWithRetryPolicy(client, policy)
	start := time.Now()
	_, err = api.GetRun(context.Background(), &gc.GetRunRequest{RunId: "test-run"})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, time.Second)
	require.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
}

func TestRetryReadDoesNotRetryRemoteDeadlineExceeded(t *testing.T) {
	calls := 0
	_, err := retryRead(
		context.Background(),
		"TestRetryRead",
		testRetryPolicy(),
		func(context.Context) (string, error) {
			calls++
			return "", status.Error(codes.DeadlineExceeded, "remote deadline")
		},
	)

	require.Error(t, err)
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Equal(t, 1, calls)
}

func TestRetryReadEnforcesOverallTimeoutAcrossAttempts(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 200 * time.Millisecond
	policy.TotalTimeout = 70 * time.Millisecond
	policy.MaxAttempts = 5
	policy.InitialBackoff = 5 * time.Millisecond
	policy.MaxBackoff = 5 * time.Millisecond

	calls := 0
	start := time.Now()
	_, err := retryRead(
		context.Background(),
		"TestRetryRead",
		policy,
		func(ctx context.Context) (string, error) {
			calls++
			<-ctx.Done()
			return "", ctx.Err()
		},
	)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, time.Second)
	require.Equal(t, 1, calls)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRetryReadRejectsSuccessAfterAttemptTimeout(t *testing.T) {
	policy := testRetryPolicy()
	policy.AttemptTimeout = 10 * time.Millisecond
	policy.TotalTimeout = time.Second
	policy.MaxAttempts = 1

	result, err := retryRead(
		context.Background(),
		"TestLateSuccess",
		policy,
		func(ctx context.Context) (string, error) {
			<-ctx.Done()
			return "late success", nil
		},
	)

	require.Empty(t, result)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
