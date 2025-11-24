// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package artifactclient provides client functionality for reading pipeline artifacts
// using HTTP streaming.
package artifactclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client/tokenrefresher"
)

// ReadArtifactRequest represents a request to read artifact content
type ReadArtifactRequest struct {
	RunID        string
	NodeID       string
	ArtifactName string
}

// String returns a string representation for use as a map key
func (r *ReadArtifactRequest) String() string {
	return r.RunID + "/" + r.NodeID + "/" + r.ArtifactName
}

// ReadArtifactResponse contains the artifact content data
type ReadArtifactResponse struct {
	Data []byte
}

// ErrorCode represents different types of errors
type ErrorCode int

const (
	// ErrorCodePermanent indicates errors that should not be retried
	ErrorCodePermanent ErrorCode = iota
	// ErrorCodeTransient indicates errors that may be retried
	ErrorCodeTransient
)

// Error represents an artifact operation error
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// NewError creates a new artifact error
func NewError(code ErrorCode, cause error, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// Client defines the interface for artifact operations
type Client interface {
	ReadArtifact(request *ReadArtifactRequest) (*ReadArtifactResponse, error)
}

// client handles artifact operations using HTTP streaming
type client struct {
	httpBaseURL    string
	httpClient     *http.Client
	tokenRefresher *tokenrefresher.TokenRefresher
}

var _ Client = &client{}

// NewClient creates a new artifact client
func NewClient(httpBaseURL string, httpClient *http.Client, tokenRefresher *tokenrefresher.TokenRefresher) Client {
	return &client{
		httpBaseURL:    httpBaseURL,
		httpClient:     httpClient,
		tokenRefresher: tokenRefresher,
	}
}

// ReadArtifact reads artifact content using HTTP streaming.
//
// Error Handling:
// - Returns nil for artifacts that don't exist (HTTP 404)
// - Returns CUSTOM_CODE_PERMANENT for client errors (400, 403) and unexpected failures
// - Returns CUSTOM_CODE_TRANSIENT for retryable errors (401, 500, network issues)
// - Automatically refreshes tokens on expiry; callers should retry transient errors
func (a *client) ReadArtifact(request *ReadArtifactRequest) (*ReadArtifactResponse, error) {
	url := fmt.Sprintf("%s/apis/v1beta1/runs/%s/nodes/%s/artifacts/%s:read",
		a.httpBaseURL, request.RunID, request.NodeID, request.ArtifactName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewError(ErrorCodePermanent, err,
			"Failed to create HTTP request: %v", err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+a.tokenRefresher.GetToken())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "service account token has expired") {
			// If unauthenticated because SA token is expired, refresh token and return transient error
			if refreshErr := a.tokenRefresher.RefreshToken(); refreshErr != nil {
				return nil, NewError(ErrorCodePermanent, refreshErr,
					"Failed to refresh token: %v", refreshErr.Error())
			}
			return nil, NewError(ErrorCodeTransient, err,
				"Error while reading artifact due to token expiry: %v", err.Error())
		}
		return nil, NewError(ErrorCodePermanent, err,
			"Failed to make HTTP request: %v", err.Error())
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			glog.Warningf("Failed to close response body: %v", closeErr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, NewError(ErrorCodePermanent, err,
				"Failed to read response body: %v", err.Error())
		}

		var jsonResponse struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(body, &jsonResponse); err != nil {
			return nil, NewError(ErrorCodePermanent, err,
				"Failed to parse JSON response: %v", err.Error())
		}

		decodedData, err := base64.StdEncoding.DecodeString(jsonResponse.Data)
		if err != nil {
			return nil, NewError(ErrorCodePermanent, err,
				"Failed to decode base64 data: %v", err.Error())
		}

		return &ReadArtifactResponse{Data: decodedData}, nil

	case http.StatusNotFound:
		return nil, nil

	case http.StatusUnauthorized:
		// Unauthorized - refresh token and return transient error
		if refreshErr := a.tokenRefresher.RefreshToken(); refreshErr != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				glog.Warningf("Failed to close response body: %v", closeErr)
			}
			return nil, NewError(ErrorCodePermanent, refreshErr,
				"Failed to refresh token: %v", refreshErr.Error())
		}
		return nil, NewError(ErrorCodeTransient, fmt.Errorf("HTTP 401"),
			"Failed to read artifact, unauthorized (token may have expired)")

	case http.StatusForbidden:
		return nil, NewError(ErrorCodePermanent, fmt.Errorf("HTTP 403"),
			"Failed to read artifact, forbidden")

	case http.StatusBadRequest:
		return nil, NewError(ErrorCodePermanent, fmt.Errorf("HTTP 400"),
			"Failed to read artifact, bad request")

	case http.StatusInternalServerError:
		return nil, NewError(ErrorCodeTransient, fmt.Errorf("HTTP 500"),
			"Failed to read artifact, internal server error")

	default:
		return nil, NewError(ErrorCodePermanent, fmt.Errorf("HTTP %d", resp.StatusCode),
			"Failed to read artifact, HTTP status: %d", resp.StatusCode)
	}
}
