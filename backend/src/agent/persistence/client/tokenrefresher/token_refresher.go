// Package tokenrefresher provides functionality to periodically refresh service account tokens
// from the filesystem for secure authentication in Kubernetes environments.
package tokenrefresher

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const SaTokenFile = "/var/run/secrets/kubeflow/tokens/persistenceagent-sa-token"

type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

type TokenRefresher struct {
	mu         sync.RWMutex
	interval   *time.Duration
	token      string
	fileReader *FileReader
}

type FileReaderImpl struct{}

func (r *FileReaderImpl) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func NewTokenRefresher(interval time.Duration, fileReader FileReader) *TokenRefresher {
	if fileReader == nil {
		fileReader = &FileReaderImpl{}
	}

	tokenRefresher := &TokenRefresher{
		interval:   &interval,
		fileReader: &fileReader,
	}

	return tokenRefresher
}

func (tr *TokenRefresher) StartTokenRefreshTicker() error {
	err := tr.RefreshToken()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(*tr.interval)
	go func() {
		for range ticker.C {
			if err := tr.RefreshToken(); err != nil {
				log.Errorf("Failed to refresh token: %v", err)
			}
		}
	}()
	return nil
}

func (tr *TokenRefresher) GetToken() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.token
}

func (tr *TokenRefresher) RefreshToken() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	b, err := (*tr.fileReader).ReadFile(SaTokenFile)
	if err != nil {
		log.Errorf("Error reading persistence agent service account token '%s': %v", SaTokenFile, err)
		return err
	}
	tr.token = string(b)
	return nil
}
