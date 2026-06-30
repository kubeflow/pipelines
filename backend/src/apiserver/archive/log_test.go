// Copyright 2020 The Kubeflow Authors
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

package archive

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func compressInput(t *testing.T, content string) []byte {
	src := bytes.Buffer{}
	gw := gzip.NewWriter(&src)
	_, err := gw.Write([]byte(content))
	assert.Nil(t, err)
	err = gw.Close()
	assert.Nil(t, err)
	return src.Bytes()
}

func compressTarInput(t *testing.T, name string, content string) []byte {
	src := bytes.Buffer{}
	gw := gzip.NewWriter(&src)
	tw := tar.NewWriter(gw)
	contents := []byte(content)
	err := tw.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0600,
		Size:     int64(len(contents)),
		Typeflag: tar.TypeReg,
	})
	require.Nil(t, err)
	_, err = tw.Write(contents)
	require.Nil(t, err)
	assert.Nil(t, tw.Close())
	assert.Nil(t, gw.Close())
	return src.Bytes()
}

type oneByteReader struct {
	content []byte
	offset  int
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.offset >= len(r.content) {
		return 0, io.EOF
	}
	p[0] = r.content[r.offset]
	r.offset++
	return 1, nil
}

var logJsonLines = `
{"timestamp": "2020-08-31T15:00:00Z", "log": "[INFO] OK"}
{"log": "[ERROR] Unable to connect"}
`

var logText = `
2020-08-31T15:00:00Z [INFO] OK
[ERROR] Unable to connect
`

var logCriOText = `
2020-08-31T15:00:00.000000000Z stdout F [INFO] OK
2020-08-31T15:00:02.260657206Z stderr F [ERROR] Unable to connect
`

var (
	logTs0, _ = time.Parse(time.RFC3339, "2020-08-31T15:00:00Z")
	logTs1, _ = time.Parse(time.RFC3339, "2020-08-31T15:00:02.260657206Z")
)

func initLogArchive() *LogArchive {
	return NewLogArchive("/logs", "main.log")
}

func TestGetLogObjectKey(t *testing.T) {
	logArchive := initLogArchive()
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})

	key, err := logArchive.GetLogObjectKey(workflow, "node-id-98765432")
	assert.Nil(t, err)
	assert.Equal(t, "/logs/MY_NAME/node-id-98765432/main.log", key)
}

func TestGetLogObjectKey_InvalidConfig(t *testing.T) {
	logArchive := NewLogArchive("", "")
	_, err := logArchive.GetLogObjectKey(nil, "node-id-98765432")
	assert.NotNil(t, err)
}

func TestCopyLogFromArchive_FromJsonToJson(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatJSON}
	dst := bytes.Buffer{}
	src := compressInput(t, logJsonLines)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	var entry RunLogEntry
	assert.True(t, scanner.Scan())
	line := scanner.Bytes()
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.Equal(t, logTs0, entry.Timestamp)
	assert.Equal(t, "[INFO] OK", entry.Log)

	assert.True(t, scanner.Scan())
	line = scanner.Bytes()
	entry = RunLogEntry{}
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.True(t, entry.Timestamp.IsZero())
	assert.Equal(t, "[ERROR] Unable to connect", entry.Log)
}

func TestCopyLogFromArchive_FromJsonToText(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: false}
	dst := bytes.Buffer{}
	src := compressInput(t, logJsonLines)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "[INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestCopyLogFromArchiveReader_AllowsLargeLogLine(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: false}
	dst := bytes.Buffer{}
	largeLogLine := strings.Repeat("x", bufio.MaxScanTokenSize*2)
	src := compressInput(t, largeLogLine+"\n")

	err := logArchive.CopyLogFromArchiveReader(bytes.NewReader(src), &dst, opts)
	require.Nil(t, err)
	assert.Equal(t, largeLogLine+"\n", dst.String())
}

func TestCopyLogFromArchiveReader_FromTarGzipStream(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: false}
	dst := bytes.Buffer{}
	src := compressTarInput(t, "main.log", logJsonLines)

	err := logArchive.CopyLogFromArchiveReader(&oneByteReader{content: src}, &dst, opts)
	require.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	require.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "[INFO] OK", line)

	require.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestOneByteReaderAllowsZeroLengthRead(t *testing.T) {
	reader := &oneByteReader{content: []byte("x")}
	buffer := make([]byte, 0)

	n, err := reader.Read(buffer)

	assert.Equal(t, 0, n)
	require.Nil(t, err)
	assert.Equal(t, 0, reader.offset)
}

func TestCopyLogFromArchive_FromJsonToTextWithTimestamp(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: true}
	dst := bytes.Buffer{}
	src := compressInput(t, logJsonLines)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "2020-08-31T15:00:00Z [INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestCopyLogFromArchive_FromTextToJson(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatJSON}
	dst := bytes.Buffer{}
	src := compressInput(t, logText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	var entry RunLogEntry
	assert.True(t, scanner.Scan())
	line := scanner.Bytes()
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.Equal(t, logTs0, entry.Timestamp)
	assert.Equal(t, "[INFO] OK", entry.Log)

	assert.True(t, scanner.Scan())
	line = scanner.Bytes()
	entry = RunLogEntry{}
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.True(t, entry.Timestamp.IsZero())
	assert.Equal(t, "[ERROR] Unable to connect", entry.Log)
}

func TestCopyLogFromArchive_FromTextToText(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: false}
	dst := bytes.Buffer{}
	src := compressInput(t, logText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "[INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestCopyLogFromArchive_FromTextToTextWithTimestamp(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: true}
	dst := bytes.Buffer{}
	src := compressInput(t, logText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "2020-08-31T15:00:00Z [INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestCopyLogFromArchive_FromCriOTextToJson(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatJSON}
	dst := bytes.Buffer{}
	src := compressInput(t, logCriOText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	var entry RunLogEntry
	assert.True(t, scanner.Scan())
	line := scanner.Bytes()
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.Equal(t, logTs0, entry.Timestamp)
	assert.Equal(t, "[INFO] OK", entry.Log)

	assert.True(t, scanner.Scan())
	line = scanner.Bytes()
	entry = RunLogEntry{}
	err = json.Unmarshal(line, &entry)
	assert.Nil(t, err)
	assert.Equal(t, logTs1, entry.Timestamp)
	assert.Equal(t, "[ERROR] Unable to connect", entry.Log)
}

func TestCopyLogFromArchive_FromCriOTextToText(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: false}
	dst := bytes.Buffer{}
	src := compressInput(t, logCriOText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "[INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "[ERROR] Unable to connect", line)
}

func TestCopyLogFromArchive_FromCriOTextToTextWithTimestamp(t *testing.T) {
	logArchive := initLogArchive()
	opts := ExtractLogOptions{LogFormat: LogFormatText, Timestamps: true}
	dst := bytes.Buffer{}
	src := compressInput(t, logCriOText)

	err := logArchive.CopyLogFromArchive(src, &dst, opts)
	assert.Nil(t, err)

	scanner := bufio.NewScanner(&dst)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.Equal(t, "2020-08-31T15:00:00.000000000Z [INFO] OK", line)

	assert.True(t, scanner.Scan())
	line = scanner.Text()
	assert.Equal(t, "2020-08-31T15:00:02.260657206Z [ERROR] Unable to connect", line)
}
