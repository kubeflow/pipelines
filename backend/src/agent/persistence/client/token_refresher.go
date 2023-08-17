package client

import (
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

type TokenRefresherInterface interface {
	GetToken() string
	RefreshToken() error
}

const SaTokenFile = "/var/run/secrets/kubeflow/tokens/persistenceagent-sa-token"

type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

type tokenRefresher struct {
	mu         sync.RWMutex
	interval   *time.Duration
	token      string
	fileReader *FileReader
}

type FileReaderImpl struct{}

func (r *FileReaderImpl) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func NewTokenRefresher(interval time.Duration, fileReader FileReader) *tokenRefresher {
	if fileReader == nil {
		fileReader = &FileReaderImpl{}
	}

	tokenRefresher := &tokenRefresher{
		interval:   &interval,
		fileReader: &fileReader,
	}

	return tokenRefresher
}

func (tr *tokenRefresher) StartTokenRefreshTicker() error {
	err := tr.RefreshToken()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(*tr.interval)
	go func() {
		for range ticker.C {
			tr.RefreshToken()
		}
	}()
	return err
}

func (tr *tokenRefresher) GetToken() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.token
}

func (tr *tokenRefresher) RefreshToken() error {
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
