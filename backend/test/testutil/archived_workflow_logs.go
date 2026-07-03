package testutil

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	pollTimeout  = 90 * time.Second
	pollInterval = 2 * time.Second

	maxBytesPerWorkflow = 200_000
	maxBytesPerStep     = 80_000

	ArchivedWorkflowLogsReportTitle = "Argo Workflows archived logs"
)

func getMinioCreds(k8Client *kubernetes.Clientset, namespace string) (string, string, error) {
	secret, err := k8Client.CoreV1().Secrets(namespace).Get(context.Background(), *config.MinioSecretName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	access := strings.TrimSpace(string(secret.Data["accesskey"]))
	secretKey := strings.TrimSpace(string(secret.Data["secretkey"]))
	if access == "" || secretKey == "" {
		return "", "", fmt.Errorf("missing accesskey/secretkey in secret %s/%s", namespace, *config.MinioSecretName)
	}
	return access, secretKey, nil
}

func newMinioClient(accessKey, secretKey string) (*minio.Client, error) {
	return minio.New(*config.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
}

func listLogKeys(ctx context.Context, client *minio.Client, bucket, prefix string) ([]string, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}
	var keys []string
	for obj := range client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		if strings.HasSuffix(obj.Key, ".log") {
			keys = append(keys, obj.Key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func pollForLogKeys(ctx context.Context, client *minio.Client, bucket, prefix string) ([]string, time.Duration, error) {
	start := time.Now()
	deadline := start.Add(pollTimeout)
	for {
		keys, err := listLogKeys(ctx, client, bucket, prefix)
		if err == nil && len(keys) > 0 {
			return keys, time.Since(start), nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return nil, time.Since(start), fmt.Errorf("timed out listing objects under %q: %w", prefix, err)
			}
			return nil, time.Since(start), nil
		}
		time.Sleep(pollInterval)
	}
}

func tailObject(ctx context.Context, client *minio.Client, bucket, key string, maxBytes int) (string, error) {
	stat, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return "", err
	}
	size := stat.Size
	if size <= 0 {
		return "", nil
	}
	start := int64(0)
	if size > int64(maxBytes) {
		start = size - int64(maxBytes)
	}
	var opts minio.GetObjectOptions
	if err := opts.SetRange(start, size-1); err != nil {
		return "", err
	}
	obj, err := client.GetObject(ctx, bucket, key, opts)
	if err != nil {
		return "", err
	}
	b, readErr := io.ReadAll(obj)
	closeErr := obj.Close()
	if readErr != nil {
		return "", readErr
	}
	if closeErr != nil {
		return "", fmt.Errorf("failed to close S3 object reader for %s/%s: %w", bucket, key, closeErr)
	}
	out := string(b)
	if start > 0 {
		out = fmt.Sprintf("[truncated: showing last %d bytes of %d]\n%s", maxBytes, size, out)
	}
	return out, nil
}

func tryAppendWorkflowKeysOrMessage(
	lines *[]string,
	ctx context.Context,
	client *minio.Client,
	workflowNamespace string,
	workflow string,
) (keys []string, ok bool) {
	prefix := fmt.Sprintf(*config.MinioLogsPrefixFmt, workflowNamespace)
	wfPrefix := fmt.Sprintf("%s/%s/", prefix, workflow)
	bucket := *config.MinioBucket
	keys, waited, err := pollForLogKeys(ctx, client, bucket, wfPrefix)
	if err != nil {
		*lines = append(*lines, fmt.Sprintf("[error waiting for *.log under s3://%s/%s after %s] %v", bucket, wfPrefix, waited, err))
		return nil, false
	}
	if len(keys) == 0 {
		*lines = append(*lines, fmt.Sprintf("[no *.log files found under s3://%s/%s after waiting %s]", bucket, wfPrefix, waited))
		return nil, false
	}
	return keys, true
}

func buildWorkflowLogsText(ctx context.Context, client *minio.Client, workflowNamespace string, workflows []string) string {
	lines := []string{"===== Argo Workflows archived logs (tailed) ====="}
	bucket := *config.MinioBucket
	for _, wf := range workflows {
		lines = append(lines, fmt.Sprintf("--- Workflow: %s ---", wf))
		keys, ok := tryAppendWorkflowKeysOrMessage(&lines, ctx, client, workflowNamespace, wf)
		if !ok {
			continue
		}

		bytesBudget := maxBytesPerWorkflow
		for _, key := range keys {
			if bytesBudget <= 0 {
				lines = append(lines, fmt.Sprintf("[truncated: workflow %s exceeded max bytes budget]", wf))
				break
			}
			step := path.Base(path.Dir(key))
			lines = append(lines, fmt.Sprintf("[step: %s]", step))
			maxBytes := min(maxBytesPerStep, bytesBudget)
			content, err := tailObject(ctx, client, bucket, key, maxBytes)
			if err != nil {
				lines = append(lines, fmt.Sprintf("[failed to fetch %s] %v", key, err))
				lines = append(lines, "")
				continue
			}
			lines = append(lines, strings.TrimRight(content, "\n"))
			lines = append(lines, "")
			bytesBudget -= len([]byte(content))
		}
	}
	lines = append(lines, "===== End Argo Workflows logs =====")
	return strings.Join(lines, "\n") + "\n"
}

func BuildArchivedWorkflowLogsReport(k8Client *kubernetes.Clientset, runIDs []string) (string, error) {
	if k8Client == nil {
		return "WARNING: k8s client is nil; skipping archived workflow logs\n", nil
	}
	if len(runIDs) == 0 {
		return "WARNING: no run IDs provided; skipping archived workflow logs\n", nil
	}
	workflowNamespace := GetNamespace()
	if workflowNamespace == "" {
		return "WARNING: workflow namespace is empty; skipping archived workflow logs\n", nil
	}

	deployNamespace := *config.Namespace
	accessKey, secretKey, err := getMinioCreds(k8Client, deployNamespace)
	if err != nil {
		return fmt.Sprintf("WARNING: failed to read %s secret (%s/%s): %v\n", *config.MinioSecretName, deployNamespace, *config.MinioSecretName, err), err
	}
	client, err := newMinioClient(accessKey, secretKey)
	if err != nil {
		return fmt.Sprintf("WARNING: failed to create S3 client: %v\n", err), err
	}

	workflowSet := make(map[string]struct{})
	for _, runID := range runIDs {
		wf := GetWorkflowNameByRunID(workflowNamespace, runID)
		if wf != "" {
			workflowSet[wf] = struct{}{}
		}
	}
	if len(workflowSet) == 0 {
		return "WARNING: no workflows found for run IDs; skipping\n", nil
	}
	var workflows []string
	for wf := range workflowSet {
		workflows = append(workflows, wf)
	}
	sort.Strings(workflows)

	logger.Log("Fetching archived workflow logs for %d workflow(s) in namespace %q", len(workflows), workflowNamespace)
	text := buildWorkflowLogsText(context.Background(), client, workflowNamespace, workflows)
	return text, nil
}
