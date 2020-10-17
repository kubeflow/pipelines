// Copyright 2020 Google LLC
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
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type LogFormat string

const (
	LogFormatJSON = LogFormat("json-lines")
	LogFormatText = LogFormat("text")
)

type ExtractLogOptions struct {
	LogFormat  LogFormat
	Timestamps bool
}

type LogArchiveInterface interface {
	GetLogObjectKey(workflow *util.Workflow, nodeId string) (string, error)
	CopyLogFromArchive(logContent []byte, dst io.Writer, opts ExtractLogOptions) error
}

// Log Archive
type RunLogEntry struct {
	Log       string    `json:"log"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// Inspired on fluent-bit parsers
// https://github.com/fluent/fluent-bit/blob/master/tests/runtime/data/kubernetes/parsers.conf
// Apache License, Version 2.0, January 2004
var k8sLogPrefixExp = regexp.MustCompile(`(?m)^(\d{4}-\d{2}-\d{2}T\S+)\s(.+)$`)
var crioLogPrefixExp = regexp.MustCompile(`(?m)^(.+)\s(stdout|stderr)\s\w\s(.+)$`)

type LogArchive struct {
	logFileName   string
	logPathPrefix string
}

func NewLogArchive(logPathPrefix, logFileName string) *LogArchive {
	return &LogArchive{
		logFileName:   logFileName,
		logPathPrefix: logPathPrefix,
	}
}

func (a *LogArchive) GetLogObjectKey(workflow *util.Workflow, nodeID string) (key string, err error) {
	if a.logPathPrefix == "" || a.logFileName == "" || workflow == nil {
		err = fmt.Errorf("invalid log archive configuration: %v", a)
	} else {
		key = strings.Join([]string{a.logPathPrefix, workflow.Name, nodeID, a.logFileName}, "/")
	}
	return
}

// CopyLogFromArchive copies a task run archived log into expected format.
func (a *LogArchive) CopyLogFromArchive(logContent []byte, dst io.Writer, opts ExtractLogOptions) error {
	reader, err := decompressLogArchive(logContent)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// line := strings.Trim(scanner.LogFormatText(), "\n\r\t ")
		bytes := scanner.Bytes()
		if len(bytes) == 0 {
			continue
		}
		var err error
		var entry RunLogEntry
		if json.Unmarshal(bytes, &entry) == nil {
			if opts.LogFormat == LogFormatJSON {
				err = writeBytesLn(dst, bytes)
			} else if opts.Timestamps && !entry.Timestamp.IsZero() {
				_, err = fmt.Fprintf(dst, "%s %s\n", entry.Timestamp.Format(time.RFC3339), entry.Log)
			} else {
				_, err = fmt.Fprintln(dst, entry.Log)
			}
		} else if result := crioLogPrefixExp.FindSubmatch(bytes); result != nil && len(result) == 4 {
			err = writeLogLn(dst, result[3], result[1], opts)
		} else if result := k8sLogPrefixExp.FindSubmatch(bytes); result != nil && len(result) == 3 {
			err = writeLogLn(dst, result[2], result[1], opts)
		} else {
			err = writeLogLn(dst, bytes, nil, opts)
		}
		if err != nil {
			return util.NewInternalServerError(err, "error in parsing the log lines")
		}
	}

	return nil
}

func decompressLogArchive(logContent []byte) (reader io.Reader, err error) {
	// Decompress tar archive
	compressedReader := bytes.NewReader(logContent)
	decompressedLogs, gzipErr := gzip.NewReader(compressedReader)
	if gzipErr != nil {
		// err = util.NewInternalServerError(gzipErr, "Failed to decompress the archived log file")
		// It's not compressed - use original content
		reader = bytes.NewReader(logContent)
		return
	}

	archiveReader := tar.NewReader(decompressedLogs)
	header, tarErr := archiveReader.Next()
	if tarErr != nil || header.Typeflag != tar.TypeReg {
		// It's not a tar archive - use decompressed content
		compressedReader.Reset(logContent)
		decompressedLogs.Reset(compressedReader)
		reader = decompressedLogs
	} else {
		reader = archiveReader
	}
	return
}

func writeLogLn(dst io.Writer, log []byte, timestamp []byte, opts ExtractLogOptions) (err error) {
	if opts.LogFormat == LogFormatJSON {
		var ts time.Time
		if timestamp != nil {
			ts, _ = time.Parse(time.RFC3339, string(timestamp))
		}
		entry := RunLogEntry{Timestamp: ts, Log: string(log)}
		if m, err := json.Marshal(entry); err == nil {
			err = writeBytesLn(dst, m)
		}
	} else if opts.Timestamps && timestamp != nil {
		err = writeBytesLn(dst, timestamp, []byte{' '}, log)
	} else {
		err = writeBytesLn(dst, log)
	}
	return
}

func writeBytesLn(dst io.Writer, bs ...[]byte) (err error) {
	for _, b := range bs {
		_, err = dst.Write(b)
		if err != nil {
			return
		}
	}
	_, err = dst.Write([]byte{'\n'})
	return
}
