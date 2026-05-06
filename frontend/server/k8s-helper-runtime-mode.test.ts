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

import { afterEach, describe, expect, it, vi } from 'vitest';

describe('k8s-helper runtime mode gating', () => {
  afterEach(() => {
    delete process.env.KFP_V2_RUNTIME_MODE;
    delete process.env.KFP_V2_RUNTIME_EXECUTOR;
    vi.resetModules();
  });

  it('does not initialize kubernetes clients during import in coordinator docker mode', async () => {
    process.env.KFP_V2_RUNTIME_MODE = 'coordinator-poc';
    process.env.KFP_V2_RUNTIME_EXECUTOR = 'docker';

    const module = await import('./k8s-helper.js');

    expect(module.TEST_ONLY.kubernetesClientsEnabled()).toBe(false);
    expect(module.TEST_ONLY.kubernetesClientsInitialized()).toBe(false);
  });

  it('treats an unset executor as docker when not running in kubernetes', async () => {
    process.env.KFP_V2_RUNTIME_MODE = 'coordinator-poc';

    const module = await import('./k8s-helper.js');

    expect(module.TEST_ONLY.kubernetesClientsEnabled()).toBe(false);
    expect(module.TEST_ONLY.kubernetesClientsInitialized()).toBe(false);
  });
});
