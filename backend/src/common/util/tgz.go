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

// SaturatingAdd adds two int64s and returns math.MaxInt64 on overflow.
func SaturatingAdd(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

// ArchiveTraversalBudget returns an overflow-safe decompressed-byte budget for
// scanning archive headers and metadata (e.g. PAX/GNU) surrounding an entry.
// The budget is maxFileSize plus a 1 MiB overhead for framing, saturating
// at math.MaxInt64 - 1 to avoid wrapping negative when callers add a sentinel byte.
func ArchiveTraversalBudget(maxFileSize int64) int64 {
	const overhead = 1 << 20 // 1 MiB for archive headers and metadata
	if maxFileSize > math.MaxInt64-overhead-1 {
		return math.MaxInt64 - 1
	}
	return maxFileSize + overhead
}

// SaturatingMultiply multiplies two int64s and returns math.MaxInt64 on overflow.
func SaturatingMultiply(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > math.MaxInt64/b {
		return math.MaxInt64
	}
	return a * b
}

// ArchiveWireResponseBudget computes a safe upper bound for an encoded artifact
// HTTP body. The calculation is intentionally conservative:
//
//  1. Traversal budget: ArchiveTraversalBudget(maxFileSize) bytes uncompressed.
//  2. DEFLATE stored-block framing: 5 bytes per 65535-byte block (worst case for
//     gzip.NoCompression).
//  3. gzip header/trailer: 18 mandatory bytes plus up to 70000 bytes for optional
//     fields (RFC 1952 Extra up to 65535 bytes, FNAME, FCOMMENT, CRC16 header).
//  4. Base64 expansion: ceil(compressed / 3) * 4.
//  5. JSON framing: {"data":""} is 11 bytes.
//
// Policy: This serves as an independent finite wire limit for security. It explicitly
// bounds canonical gzip.NoCompression outputs with up to 70000 bytes of optional headers.
// It is not mathematically guaranteed to bound every conceivable valid gzip stream
// (for example, pathological streams with arbitrary zero-output DEFLATE blocks via
// excessive Flush calls or massive numbers of concatenated members can legitimately
// exceed this cap). Callers must enforce this as a hard wire limit.
func ArchiveWireResponseBudget(maxFileSize int64) int64 {
	// Maximum decompressed traversal budget.
	budget := ArchiveTraversalBudget(maxFileSize)

	// DEFLATE stored-block overhead: 5 bytes per 65535-byte block.
	numBlocks := SaturatingAdd(budget/65535, 1)
	deflateStoredOverhead := SaturatingMultiply(numBlocks, 5)

	// gzip framing: 18 mandatory bytes + up to 70000 bytes for optional header fields
	// (Extra field max 65535, plus FNAME/FCOMMENT/CRC16 slack of ~4465 bytes).
	const gzipFramingAllowance = 70018

	deflateOverhead := SaturatingAdd(deflateStoredOverhead, gzipFramingAllowance)

	// Worst-case compressed size.
	maxCompressed := SaturatingAdd(budget, deflateOverhead)

	// Base64 encoding size: 4 * ceil(maxCompressed / 3).
	var maxBase64 int64
	if maxCompressed > (math.MaxInt64/4)*3 {
		maxBase64 = math.MaxInt64
	} else {
		maxBase64 = ((maxCompressed + 2) / 3) * 4
	}

	// JSON framing: {"data":""} is 11 bytes.
	return SaturatingAdd(maxBase64, 11)
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

	limitedReader := &io.LimitedReader{R: tr, N: SaturatingAdd(maxFileSize, 1)}
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
