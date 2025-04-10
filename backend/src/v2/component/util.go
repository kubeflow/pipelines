package component

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"io"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

// CopyThisBinary copies the running binary into destination path.
func CopyThisBinary(destination string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("copy this binary to %s: %w", destination, err)
		}
	}()

	path, err := findThisBinary()
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o555) // 0o555 -> readable and executable by all
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Close()
}

func findThisBinary() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("findThisBinary failed: %w", err)
	}
	return path, nil
}

type K8sClientProvider func() (kubernetes.Interface, error)

var defaultK8sClientProvider K8sClientProvider = func() (kubernetes.Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

type MetadataClientProvider func(address string, port string) (metadata.ClientInterface, error)

var defaultMetadataClientProvider MetadataClientProvider = func(address string, port string) (metadata.ClientInterface, error) {
	return metadata.NewClient(address, port)
}

type CacheClientProvider func() (*cacheutils.Client, error)

var defaultCacheClientProvider CacheClientProvider = func() (*cacheutils.Client, error) {
	return cacheutils.NewClient()
}

type CreateFileFunc func(string) (*os.File, error)
