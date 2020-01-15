// Copyright 2019 Google LLC
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
import * as os from 'os';
import { loadConfigs } from './configs';

describe('loadConfigs', () => {
  it('should throw error if no static dir provided', () => {
    const argv = ['node', 'dist/server.js'];
    expect(() => loadConfigs(argv, {})).toThrowError();
  });

  it('default port should be 3000', () => {
    const tmpdir = os.tmpdir();
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {});
    expect(configs.server.port).toBe(3000);
    expect(configs.server.staticDir).toBe(tmpdir);
  });
});
