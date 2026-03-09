// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { isAllowedDomain } from './domain-checker.js';

describe('isAllowedDomain', () => {
  it('matches a host-only allowlist for a plain https URL', () => {
    expect(isAllowedDomain('https://example.com/path/to/artifact', '^example\\.com$')).toBe(true);
  });

  it('matches a host-only allowlist when the URL contains user info', () => {
    expect(isAllowedDomain('https://user@example.com/path/to/artifact', '^example\\.com$')).toBe(
      true,
    );
  });

  it('rejects a URL whose host does not match the allowlist', () => {
    expect(isAllowedDomain('https://evil.example/path/to/artifact', '^example\\.com$')).toBe(false);
  });
});
