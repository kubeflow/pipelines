// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dependency

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// GO-2026-6860: ignored end tags must not cause quadratic text concatenation.
// This test-only package has no production initializers or parallel tests,
// isolating the process-wide allocation measurement from application workers.
func TestHTMLTextCoalescingAllocation(t *testing.T) {
	const tokens = 10000
	for _, prefix := range []string{"", "<table>"} {
		t.Run(prefix, func(t *testing.T) {
			input := prefix + strings.Repeat("a</x>", tokens)
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			document, err := html.Parse(strings.NewReader(input))
			runtime.ReadMemStats(&after)
			if err != nil {
				t.Fatal(err)
			}
			text := document.FirstChild.LastChild.FirstChild
			if text == nil || text.Type != html.TextNode || len(text.Data) != tokens {
				t.Fatalf("expected %d coalesced text bytes, got %v", tokens, text)
			}
			// The fixed parser stays well below this bound; the old parser
			// copies roughly 50 MB for the same 10,000 tokens.
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 5<<20 {
				t.Fatalf("HTML text coalescing allocated %d bytes, limit %d", allocated, 5<<20)
			}
		})
	}
}

// GO-2026-6866: a nested HTML integration point must not hide an unsafe
// ancestor namespace transition when serializing raw text.
func TestHTMLForeignAncestorTextEscaping(t *testing.T) {
	for _, ancestor := range []string{"mglyph", "malignmark"} {
		t.Run(ancestor, func(t *testing.T) {
			input := "<math><mtext><table><" + ancestor + "><svg><desc><style><img src=x onerror=alert(1)>"
			document, err := html.Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			var rendered bytes.Buffer
			if err := html.Render(&rendered, document); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(rendered.String(), "<style>&lt;img src=x onerror=alert(1)&gt;</style>") {
				t.Fatalf("foreign-ancestor text was not escaped: %s", rendered.String())
			}
		})
	}
}
