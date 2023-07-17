package client

import (
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

type TokenRefresherInterface interface {
	GetToken() string
	RefreshToken()
}

const SaTokenFile = "/var/run/secrets/kubeflow/tokens/persistenceagent-sa-token"

type tokenRefresher struct {
	mu      sync.RWMutex
	seconds *time.Duration
	token   string
}

func NewTokenRefresher(seconds time.Duration) *tokenRefresher {
	tokenRefresher := &tokenRefresher{
		seconds: &seconds,
	}

	return tokenRefresher
}

func (tr *tokenRefresher) StartTokenRefreshTicker(stopCh <-chan struct{}) error {
	err := tr.readToken()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(*tr.seconds)
	go func() {
		for {
			select {
			case <-stopCh:
				ticker.Stop()
				return
			case <-ticker.C:
				tr.readToken()
			}
		}
	}()
	return err
}

func (tr *tokenRefresher) GetToken() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.token
}

func (tr *tokenRefresher) RefreshToken() {
	tr.readToken()
}

func (tr *tokenRefresher) readToken() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	b, err := os.ReadFile(SaTokenFile)
	if err != nil {
		log.Errorf("Error reading persistence agent service account token '%s': %v", SaTokenFile, err)
		return err
	}
	tr.token = string(b)
	return nil
}
