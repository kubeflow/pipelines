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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
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

// readSingleFileFromTgz streams the only regular file in a tar.gz archive to
// consume. The caller provides the maximum uncompressed file size.
func readSingleFileFromTgz(tgzContent []byte, maxFileSize int64, consume func(io.Reader) error) error {
	if maxFileSize <= 0 {
		return fmt.Errorf("maximum metrics file size must be positive")
	}

	gr, err := gzip.NewReader(bytes.NewReader(tgzContent))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	hdr, err := tr.Next()
	if err == io.EOF {
		return fmt.Errorf("metrics archive must contain exactly one regular file")
	}
	if err != nil {
		return err
	}
	if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
		return fmt.Errorf("metrics archive entry %q must be a regular file", hdr.Name)
	}
	if hdr.Size < 0 {
		return fmt.Errorf("metrics archive entry %q has invalid negative size %d", hdr.Name, hdr.Size)
	}
	if hdr.Size > maxFileSize {
		return fmt.Errorf("metrics archive entry %q exceeds maximum size of %d bytes", hdr.Name, maxFileSize)
	}

	limitedReader := &io.LimitedReader{R: tr, N: maxFileSize}
	if err := consume(limitedReader); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, limitedReader); err != nil {
		return err
	}

	if _, err := tr.Next(); err == nil {
		return fmt.Errorf("metrics archive must contain exactly one file")
	} else if err != io.EOF {
		return err
	}
	return nil
}
