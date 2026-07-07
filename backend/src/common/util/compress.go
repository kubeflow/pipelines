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

package util

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
)

// GzipCompressBase64 gzip-compresses s and returns it as a base64-encoded
// string, suitable for embedding in places that only accept plain strings
// (e.g. Argo Workflow parameters, database text columns).
func GzipCompressBase64(s string) (string, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(s)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GzipDecompressBase64 reverses GzipCompressBase64. If s is empty, or is not
// valid base64/gzip data, it is returned unchanged. This keeps the function
// safe to call on data that may have been written before compression was
// introduced, or during a rolling upgrade.
func GzipDecompressBase64(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return s, nil
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return s, nil
	}
	return string(out), nil
}
