// Copyright 2019-2020 Google LLC
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
import { parseTensorboardLogDir } from './k8s-helper';

describe('k8s-helper', () => {
  describe('parseTensorboardLogDir', () => {
    const podTemplateSpec = {
      spec: {
        containers: [
          {
            volumeMounts: [
              {
                name: 'output',
                mountPath: '/data',
              },
            ],
          },
        ],
        volumes: [
          {
            name: 'output',
            hostPath: {
              path: '/data/output',
              type: 'Directory',
            },
          },
        ],
      },
    };

    it('throws for unsupported protocol', () => {
      expect(() => parseTensorboardLogDir('unsupported://path', podTemplateSpec)).toThrowError(
        'Unsupported storage path: unsupported://path',
      );
    });

    it('handles gs storage', () => {
      const logdir = 'gs://testbucket/test/key/path';
      const url = parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles gs storage with Series', () => {
      const logdir =
        'Series1:gs://testbucket/test/key/path1,Series2:gs://testbucket/test/key/path2';
      const url = parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles volume storage', () => {
      const logdir = 'volume://output/data/volume/path';
      const url = parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('/data/volume/path');
    });

    it('handles volume storage with Series', () => {
      const logdir =
        'Series1:volume://output/data/volume/path1,Series2:volume://output/data/volume/path2';
      const url = parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('Series1:/data/volume/path1,Series2:/data/volume/path2');
    });
  });
});
