package util

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestPostForm(t *testing.T) {
	providedResp := &http.Response{
		StatusCode: 200,
	}
	fakeHTTP := NewFakeHTTP(providedResp, nil)
	resp, err := fakeHTTP.PostForm("", nil)
	assert.Equal(t, providedResp, resp)
	assert.Nil(t, err)
}

func TestPostForm_Failed(t *testing.T) {
	providedResp := &http.Response{
		StatusCode: 500,
	}
	fakeHTTP := NewFakeHTTP(providedResp, errors.New("server error"))
	resp, err := fakeHTTP.PostForm("", nil)
	assert.Equal(t, providedResp, resp)
	assert.Equal(t, "server error", err.Error())
}