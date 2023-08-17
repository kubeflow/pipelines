package client

import (
	"fmt"
	"io/fs"
	"log"
	"syscall"
	"testing"
	"time"
)

const refreshInterval = 2 * time.Second

type FileReaderFake struct {
	Data        string
	Err         error
	readCounter int
}

func (m *FileReaderFake) ReadFile(filename string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	content := fmt.Sprintf("%s-%v", m.Data, m.readCounter)
	m.readCounter++
	return []byte(content), nil
}

func Test_token_refresher(t *testing.T) {
	tests := []struct {
		name           string
		baseToken      string
		wanted         string
		refreshedToken string
		err            error
	}{
		{
			name:      "TestTokenRefresher_GetToken_Success",
			baseToken: "rightToken",
			wanted:    "rightToken-0",
			err:       nil,
		},
		{
			name:      "TestTokenRefresher_GetToken_Failed_PathError",
			baseToken: "rightToken",
			wanted:    "rightToken-0",
			err:       &fs.PathError{Err: syscall.ENOENT},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup
			fakeFileReader := &FileReaderFake{
				Data: tt.baseToken,
				Err:  tt.err,
			}
			tr := NewTokenRefresher(refreshInterval, fakeFileReader)
			err := tr.StartTokenRefreshTicker()
			if err != nil {
				got, sameType := err.(*fs.PathError)
				if sameType != true {
					t.Errorf("%v(): got = %v, wanted %v", tt.name, got, tt.err)
				}
				return
			}
			if err != nil {
				log.Fatalf("Error starting Service Account Token Refresh Ticker: %v", err)
			}

			if got := tr.GetToken(); got != tt.wanted {
				t.Errorf("%v(): got %v, wanted %v", tt.name, got, tt.wanted)
			}
		})
	}
}

func TestTokenRefresher_GetToken_After_TickerRefresh_Success(t *testing.T) {
	fakeFileReader := &FileReaderFake{
		Data: "Token",
		Err:  nil,
	}
	tr := NewTokenRefresher(1*time.Second, fakeFileReader)
	err := tr.StartTokenRefreshTicker()
	if err != nil {
		log.Fatalf("Error starting Service Account Token Refresh Ticker: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	expectedToken := "Token-1"

	if got := tr.GetToken(); got != expectedToken {
		t.Errorf("%v(): got %v, wanted 'refreshed baseToken' %v", t.Name(), got, expectedToken)
	}
}

func TestTokenRefresher_GetToken_After_ForceRefresh_Success(t *testing.T) {
	fakeFileReader := &FileReaderFake{
		Data: "Token",
		Err:  nil,
	}
	tr := NewTokenRefresher(refreshInterval, fakeFileReader)
	err := tr.StartTokenRefreshTicker()
	if err != nil {
		log.Fatalf("Error starting Service Account Token Refresh Ticker: %v", err)
	}
	tr.RefreshToken()
	expectedToken := "Token-1"

	if got := tr.GetToken(); got != expectedToken {
		t.Errorf("%v(): got %v, wanted 'refreshed baseToken' %v", t.Name(), got, expectedToken)
	}
}
