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

import { describe, it } from 'vitest';
import requests from 'supertest';
import { createMockApiApp } from '../../mock-backend/mock-api-app.ts';
import { setupMockBackendTest } from './test-helper.js';

setupMockBackendTest();

describe('standalone mock API server', () => {
  it('constructs the app with the mock backend dependency graph', async () => {
    const request = requests(createMockApiApp());

    await request.get('/apis/v2beta1/healthz').expect(200);
    await request.get('/apis/v1beta1/healthz').expect(404);
    await request.get('/APIS/V2BETA1/not-found').expect(404, 'Bad request endpoint.');
    await request.get('/APPS/TENSORBOARD/PROXY/mock-token/index.js').expect(200);
  });
});
