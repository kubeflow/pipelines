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
	"math"
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

// ArchiveTraversalBudget returns an overflow-safe decompressed-byte budget for
// scanning archive headers and metadata (e.g. PAX/GNU) surrounding an entry.
// The budget is maxFileSize plus a 1 MiB overhead for framing, saturating
// at math.MaxInt64 to avoid wrapping negative on large inputs.
func ArchiveTraversalBudget(maxFileSize int64) int64 {
	const overhead = 1 << 20 // 1 MiB for archive headers and metadata
	if maxFileSize > math.MaxInt64-overhead {
		return math.MaxInt64
	}
	return maxFileSize + overhead
}

// readSingleFileFromTgz streams the only regular file in a tar.gz archive to
// consume. The caller provides the maximum uncompressed file size.
//
// The function enforces an archive-wide decompression budget to prevent
// tar-bomb attacks in which hidden PAX or GNU metadata between entries causes
// attacker-controlled unbounded work in the persistence agent before the
// per-entry size check is reached.
func readSingleFileFromTgz(tgzContent []byte, maxFileSize int64, consume func(io.Reader) error) error {
	if maxFileSize <= 0 {
		return fmt.Errorf("maximum metrics file size must be positive")
	}

	gr, err := gzip.NewReader(bytes.NewReader(tgzContent))
	if err != nil {
		return err
	}
	defer gr.Close()

	// Wrap the gzip stream in a traversal budget that accounts for PAX/GNU
	// extended headers and tar framing consumed by tar.Reader.Next().
	// Use budget+1 so that exact-boundary exhaustion is detectable as N==0
	// rather than being silently accepted as io.EOF.
	budget := ArchiveTraversalBudget(maxFileSize)
	limitedGr := &io.LimitedReader{R: gr, N: budget + 1}
	tr := tar.NewReader(limitedGr)

	hdr, err := tr.Next()
	// Check exhaustion before any error branch: if the budget ran out,
	// tar may return io.EOF or a truncated-data error depending on alignment.
	if limitedGr.N <= 0 {
		return fmt.Errorf("metrics archive traversal exceeded budget of %d bytes", budget)
	}
	if err == io.EOF {
		return fmt.Errorf("metrics archive must contain exactly one regular file")
	}
	if err != nil {
		return err
	}
	if hdr.Typeflag != tar.TypeReg {
		return fmt.Errorf("metrics archive entry %q must be a regular file", hdr.Name)
	}
	if hdr.Size < 0 {
		return fmt.Errorf("metrics archive entry %q has invalid negative size %d", hdr.Name, hdr.Size)
	}
	if hdr.Size > maxFileSize {
		return fmt.Errorf("metrics archive entry %q exceeds maximum size of %d bytes", hdr.Name, maxFileSize)
	}

	limitedReader := &io.LimitedReader{R: tr, N: maxFileSize + 1}
	if err := consume(limitedReader); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, limitedReader); err != nil {
		return err
	}

	// Confirm no second entry exists. Check exhaustion before accepting EOF
	// so that budget-aligned archives do not bypass the check silently.
	_, nextErr := tr.Next()
	if limitedGr.N <= 0 {
		return fmt.Errorf("metrics archive traversal exceeded budget of %d bytes", budget)
	}
	if nextErr == nil {
		return fmt.Errorf("metrics archive must contain exactly one regular file")
	}
	if nextErr != io.EOF {
		return nextErr
	}
	return nil
}
