// Copyright 2018 The Kubeflow Authors
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

package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsNotFound(t *testing.T) {
	assert.Equal(t, true, IsNotFound(k8errors.NewNotFound(schema.GroupResource{}, "NAME")))
	assert.Equal(t, false, IsNotFound(k8errors.NewAlreadyExists(schema.GroupResource{}, "NAME")))
}

// CustomError tests

func TestNewCustomError(t *testing.T) {
	cause := fmt.Errorf("root cause")
	customError := NewCustomError(cause, CUSTOM_CODE_TRANSIENT, "something went %s", "wrong")
	assert.Contains(t, customError.Error(), "CustomError (code: 0)")
	assert.Contains(t, customError.Error(), "something went wrong")
	assert.Contains(t, customError.Error(), "root cause")
}

func TestNewCustomErrorf(t *testing.T) {
	customError := NewCustomErrorf(CUSTOM_CODE_PERMANENT, "item %d not found", 42)
	assert.Contains(t, customError.Error(), "CustomError (code: 1)")
	assert.Contains(t, customError.Error(), "item 42 not found")
}

func TestHasCustomCode(t *testing.T) {
	t.Run("nil error returns false", func(t *testing.T) {
		assert.False(t, HasCustomCode(nil, CUSTOM_CODE_TRANSIENT))
	})

	t.Run("matching CustomError returns true", func(t *testing.T) {
		customError := NewCustomErrorf(CUSTOM_CODE_NOT_FOUND, "not found")
		assert.True(t, HasCustomCode(customError, CUSTOM_CODE_NOT_FOUND))
	})

	t.Run("non-matching code returns false", func(t *testing.T) {
		customError := NewCustomErrorf(CUSTOM_CODE_PERMANENT, "permanent")
		assert.False(t, HasCustomCode(customError, CUSTOM_CODE_NOT_FOUND))
	})

	t.Run("non-CustomError returns false", func(t *testing.T) {
		regularError := fmt.Errorf("regular error")
		assert.False(t, HasCustomCode(regularError, CUSTOM_CODE_TRANSIENT))
	})
}

// UserError constructor tests

func TestErrorConstructors(t *testing.T) {
	baseError := fmt.Errorf("base error")

	testCases := []struct {
		name               string
		constructor        func() *UserError
		expectedCode       codes.Code
		expectedExtMessage string
	}{
		{
			name: "NewInternalServerError",
			constructor: func() *UserError {
				return NewInternalServerError(baseError, "internal %s", "details")
			},
			expectedCode:       codes.Internal,
			expectedExtMessage: "Internal Server Error",
		},
		{
			name: "NewUnavailableServerError",
			constructor: func() *UserError {
				return NewUnavailableServerError(baseError, "server %s", "down")
			},
			expectedCode:       codes.Unavailable,
			expectedExtMessage: "Service unavailable",
		},
		{
			name: "NewNotFoundError",
			constructor: func() *UserError {
				return NewNotFoundError(baseError, "resource %s not found", "foo")
			},
			expectedCode:       codes.NotFound,
			expectedExtMessage: "resource foo not found",
		},
		{
			name: "NewResourceNotFoundError",
			constructor: func() *UserError {
				return NewResourceNotFoundError("Pipeline", "my-pipeline")
			},
			expectedCode:       codes.NotFound,
			expectedExtMessage: "Pipeline my-pipeline not found",
		},
		{
			name: "NewResourcesNotFoundError",
			constructor: func() *UserError {
				return NewResourcesNotFoundError("Pipelines %s and %s", "a", "b")
			},
			expectedCode:       codes.NotFound,
			expectedExtMessage: "Pipelines a and b not found",
		},
		{
			name: "NewInvalidInputError",
			constructor: func() *UserError {
				return NewInvalidInputError("field %s is invalid", "name")
			},
			expectedCode:       codes.InvalidArgument,
			expectedExtMessage: "field name is invalid",
		},
		{
			name: "NewInvalidInputErrorWithDetails",
			constructor: func() *UserError {
				return NewInvalidInputErrorWithDetails(baseError, "bad input")
			},
			expectedCode:       codes.InvalidArgument,
			expectedExtMessage: "bad input",
		},
		{
			name: "NewAlreadyExistError",
			constructor: func() *UserError {
				return NewAlreadyExistError("resource %s exists", "foo")
			},
			expectedCode:       codes.AlreadyExists,
			expectedExtMessage: "resource foo exists",
		},
		{
			name: "NewBadRequestError",
			constructor: func() *UserError {
				return NewBadRequestError(baseError, "bad request: %s", "details")
			},
			expectedCode:       codes.Aborted,
			expectedExtMessage: "bad request: details",
		},
		{
			name: "NewBadKubernetesNameError",
			constructor: func() *UserError {
				return NewBadKubernetesNameError("Pipeline")
			},
			expectedCode:       codes.Aborted,
			expectedExtMessage: "Invalid Pipeline name",
		},
		{
			name: "NewFailedPreconditionError",
			constructor: func() *UserError {
				return NewFailedPreconditionError(baseError, "precondition %s failed", "x")
			},
			expectedCode:       codes.FailedPrecondition,
			expectedExtMessage: "precondition x failed",
		},
		{
			name: "NewUnauthenticatedError",
			constructor: func() *UserError {
				return NewUnauthenticatedError(baseError, "user %s unauthenticated", "bob")
			},
			expectedCode:       codes.Unauthenticated,
			expectedExtMessage: "user bob unauthenticated",
		},
		{
			name: "NewPermissionDeniedError",
			constructor: func() *UserError {
				return NewPermissionDeniedError(baseError, "access to %s denied", "resource")
			},
			expectedCode:       codes.PermissionDenied,
			expectedExtMessage: "access to resource denied",
		},
		{
			name: "NewUnknownApiVersionError",
			constructor: func() *UserError {
				return NewUnknownApiVersionError("v2beta1", "pipeline")
			},
			expectedCode:       codes.InvalidArgument,
			expectedExtMessage: "Error using v2beta1 with string",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			userError := testCase.constructor()
			assert.Equal(t, testCase.expectedCode, userError.ExternalStatusCode())
			assert.Equal(t, testCase.expectedExtMessage, userError.ExternalMessage())
			assert.NotEmpty(t, userError.Error())
		})
	}
}

// UserError method tests

func TestUserErrorMethods(t *testing.T) {
	baseError := fmt.Errorf("internal details")
	userError := NewNotFoundError(baseError, "resource not found")

	assert.Equal(t, "resource not found", userError.ExternalMessage())
	assert.Equal(t, codes.NotFound, userError.ExternalStatusCode())
	assert.Contains(t, userError.Error(), "NotFoundError")
	assert.Contains(t, userError.Error(), "resource not found")
	assert.NotNil(t, userError.Cause())
	assert.Contains(t, userError.String(), "resource not found")
	assert.Contains(t, userError.String(), fmt.Sprintf("code: %v", codes.NotFound))
	assert.NotNil(t, userError.Unwrap())
}

func TestNewUserError(t *testing.T) {
	cause := fmt.Errorf("some error")
	userError := NewUserError(cause, "internal msg", "external msg")
	assert.Equal(t, codes.Internal, userError.ExternalStatusCode())
	assert.Contains(t, userError.ExternalMessage(), "external msg")
	assert.Contains(t, userError.Error(), "internal msg")
}

func TestNewUserErrorWithSingleMessage(t *testing.T) {
	cause := fmt.Errorf("some error")
	userError := NewUserErrorWithSingleMessage(cause, "shared message")
	assert.Contains(t, userError.ExternalMessage(), "shared message")
	assert.Contains(t, userError.Error(), "shared message")
}

func TestExtractErrorForCLI(t *testing.T) {
	t.Run("UserError in debug mode returns internal error", func(t *testing.T) {
		userError := NewInternalServerError(fmt.Errorf("secret details"), "internal info")
		result := ExtractErrorForCLI(userError, true)
		assert.Contains(t, result.Error(), "secret details")
	})

	t.Run("UserError in non-debug mode returns external message", func(t *testing.T) {
		userError := NewInternalServerError(fmt.Errorf("secret details"), "internal info")
		result := ExtractErrorForCLI(userError, false)
		assert.Equal(t, "Internal Server Error", result.Error())
	})

	t.Run("non-UserError returns original error", func(t *testing.T) {
		regularError := fmt.Errorf("plain error")
		result := ExtractErrorForCLI(regularError, false)
		assert.Equal(t, "plain error", result.Error())
	})
}

// Wrapf and Wrap tests

func TestWrapf(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, Wrapf(nil, "context: %s", "info"))
	})

	t.Run("UserError preserves type", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		wrapped := Wrapf(userError, "context: %s", "info")
		wrappedUserError, ok := wrapped.(*UserError)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, wrappedUserError.ExternalStatusCode())
		assert.Equal(t, "not found", wrappedUserError.ExternalMessage())
	})

	t.Run("regular error wraps normally", func(t *testing.T) {
		regularError := fmt.Errorf("original")
		wrapped := Wrapf(regularError, "context: %s", "info")
		assert.Contains(t, wrapped.Error(), "original")
		assert.Contains(t, wrapped.Error(), "context: info")
	})
}

func TestWrap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, Wrap(nil, "context"))
	})

	t.Run("UserError preserves type", func(t *testing.T) {
		userError := NewInvalidInputError("bad input")
		wrapped := Wrap(userError, "additional context")
		wrappedUserError, ok := wrapped.(*UserError)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, wrappedUserError.ExternalStatusCode())
		assert.Equal(t, "bad input", wrappedUserError.ExternalMessage())
	})

	t.Run("regular error wraps normally", func(t *testing.T) {
		regularError := fmt.Errorf("original")
		wrapped := Wrap(regularError, "context")
		assert.Contains(t, wrapped.Error(), "original")
		assert.Contains(t, wrapped.Error(), "context")
	})
}

// IsUserErrorCodeMatch tests

func TestIsUserErrorCodeMatch(t *testing.T) {
	t.Run("matching code returns true", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		assert.True(t, IsUserErrorCodeMatch(userError, codes.NotFound))
	})

	t.Run("non-matching code returns false", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		assert.False(t, IsUserErrorCodeMatch(userError, codes.Internal))
	})

	t.Run("non-UserError returns false", func(t *testing.T) {
		regularError := fmt.Errorf("regular")
		assert.False(t, IsUserErrorCodeMatch(regularError, codes.NotFound))
	})
}

// ToRpcStatus tests

func TestToRpcStatus(t *testing.T) {
	t.Run("with UserError", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		rpcStatus := ToRpcStatus(userError)
		assert.NotNil(t, rpcStatus)
		assert.Equal(t, int32(codes.NotFound), rpcStatus.GetCode())
	})

	t.Run("with regular error", func(t *testing.T) {
		regularError := fmt.Errorf("regular error")
		rpcStatus := ToRpcStatus(regularError)
		assert.NotNil(t, rpcStatus)
		assert.Equal(t, int32(codes.Unknown), rpcStatus.GetCode())
	})
}

// ToError tests

func TestToError(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, ToError(nil))
	})

	t.Run("with status returns error", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		rpcStatus := ToRpcStatus(userError)
		err := ToError(rpcStatus)
		assert.NotNil(t, err)
	})
}

// ToGRPCError tests

func TestToGRPCError(t *testing.T) {
	t.Run("with UserError", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		grpcError := ToGRPCError(userError)
		assert.NotNil(t, grpcError)
	})

	t.Run("with regular error", func(t *testing.T) {
		regularError := fmt.Errorf("regular error")
		grpcError := ToGRPCError(regularError)
		assert.NotNil(t, grpcError)
		assert.Contains(t, grpcError.Error(), "regular error")
	})
}

// ToGRPCStatus tests

func TestToGRPCStatus(t *testing.T) {
	t.Run("with UserError", func(t *testing.T) {
		userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
		grpcStatus := ToGRPCStatus(userError)
		assert.NotNil(t, grpcStatus)
		assert.Equal(t, codes.NotFound, grpcStatus.Code())
	})

	t.Run("with regular error", func(t *testing.T) {
		regularError := fmt.Errorf("regular error")
		grpcStatus := ToGRPCStatus(regularError)
		assert.NotNil(t, grpcStatus)
		assert.Equal(t, codes.Internal, grpcStatus.Code())
	})
}

// GRPCStatus method test

func TestGRPCStatus(t *testing.T) {
	userError := NewNotFoundError(fmt.Errorf("cause"), "not found")
	grpcStatus := userError.GRPCStatus()
	assert.NotNil(t, grpcStatus)
	assert.Equal(t, codes.NotFound, grpcStatus.Code())
	details := grpcStatus.Details()
	assert.Greater(t, len(details), 0)
}
