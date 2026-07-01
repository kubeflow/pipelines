package testutil

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestIsPermanentLogListError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "access denied code",
			err:  minio.ErrorResponse{Code: "AccessDenied"},
			want: true,
		},
		{
			name: "forbidden status",
			err:  minio.ErrorResponse{StatusCode: http.StatusForbidden},
			want: true,
		},
		{
			name: "unauthorized status",
			err:  minio.ErrorResponse{StatusCode: http.StatusUnauthorized},
			want: true,
		},
		{
			name: "wrapped credential error",
			err:  errors.Join(errors.New("list failed"), minio.ErrorResponse{Code: "InvalidAccessKeyId"}),
			want: true,
		},
		{
			name: "transient service error",
			err:  minio.ErrorResponse{Code: "InternalError", StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "ordinary error",
			err:  errors.New("temporary network failure"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermanentLogListError(tt.err)
			if got != tt.want {
				t.Fatalf("isPermanentLogListError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestPollForLogKeysWithListerReturnsImmediatelyOnPermanentError(t *testing.T) {
	calls := 0
	_, _, err := pollForLogKeysWithLister(
		context.Background(),
		nil,
		"mlpipeline",
		"private-artifacts/kubeflow-user-example-com/workflow/",
		func(ctx context.Context, client *minio.Client, bucket, prefix string) ([]string, error) {
			calls++
			if calls > 1 {
				t.Fatalf("permanent list error was retried %d times", calls)
			}
			return nil, minio.ErrorResponse{
				Code:       "AccessDenied",
				Message:    "Access Denied",
				StatusCode: http.StatusForbidden,
			}
		},
	)

	if err == nil {
		t.Fatal("pollForLogKeysWithLister() error = nil, want permanent list error")
	}
	if calls != 1 {
		t.Fatalf("lister calls = %d, want 1", calls)
	}
	if !strings.Contains(err.Error(), "failed listing objects") {
		t.Fatalf("error = %q, want failed listing objects context", err)
	}
	if !strings.Contains(err.Error(), "Access Denied") {
		t.Fatalf("error = %q, want original Access Denied error", err)
	}
}

func TestPollForLogKeysWithListerReturnsKeys(t *testing.T) {
	wantKeys := []string{"private-artifacts/ns/workflow/main.log"}

	gotKeys, _, err := pollForLogKeysWithLister(
		context.Background(),
		nil,
		"mlpipeline",
		"private-artifacts/ns/workflow/",
		func(ctx context.Context, client *minio.Client, bucket, prefix string) ([]string, error) {
			return wantKeys, nil
		},
	)

	if err != nil {
		t.Fatalf("pollForLogKeysWithLister() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("pollForLogKeysWithLister() keys = %v, want %v", gotKeys, wantKeys)
	}
}
