// Copyright 2018 Google LLC
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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"strings"
)

// Utils to archive and extract tgz (tar.gz) file.

// ArchiveTgz takes a map of files with name as key and content as value and
// tar and gzip it to a tgz content string. Nested files and directories are
// not supported.
func ArchiveTgz(files map[string]string) (string, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return "", err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExtractTgz extracts a list of files from a tgz content. The output is a map
// with file name as key and content as value. Nested files and directories are
// not supported.
func ExtractTgz(tgzContent string) (map[string]string, error) {
	sr := strings.NewReader(tgzContent)
	gr, err := gzip.NewReader(sr)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gr)

	files := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr == nil {
			continue
		}
		fileContent, err := ioutil.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files[hdr.Name] = string(fileContent)
	}
	return files, nil
}
