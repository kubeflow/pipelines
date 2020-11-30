// Copyright 2020 Google LLC
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
import { TEST_ONLY as K8S_TEST_EXPORT } from './k8s-helper';

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
              {
                name: 'artifact',
                subPath: 'pipeline1',
                mountPath: '/data1',
              },
              {
                name: 'artifact',
                subPath: 'pipeline2',
                mountPath: '/data2',
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
          {
            name: 'artifact',
            persistentVolumeClaim: {
              claimName: 'artifact_pvc',
            },
          },
        ],
      },
    };

    it('handles not volume storage', () => {
      const logdir = 'gs://testbucket/test/key/path';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles not volume storage with Series', () => {
      const logdir =
        'Series1:gs://testbucket/test/key/path1,Series2:gs://testbucket/test/key/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles volume storage without subPath', () => {
      const logdir = 'volume://output';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('/data');
    });

    it('handles volume storage without subPath with Series', () => {
      const logdir = 'Series1:volume://output/volume/path1,Series2:volume://output/volume/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('Series1:/data/volume/path1,Series2:/data/volume/path2');
    });

    it('handles volume storage with subPath', () => {
      const logdir = 'volume://artifact/pipeline1';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('/data1');
    });

    it('handles volume storage with subPath with Series', () => {
      const logdir =
        'Series1:volume://artifact/pipeline1/path1,Series2:volume://artifact/pipeline2/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('Series1:/data1/path1,Series2:/data2/path2');
    });

    it('handles volume storage without subPath throw volume not configured error', () => {
      const logdir = 'volume://other/path';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://other/path" in pod "unknown": volume "other" not configured',
      );
    });

    it('handles volume storage without subPath throw volume not configured error with Series', () => {
      const logdir = 'Series1:volume://output/volume/path1,Series2:volume://other/volume/path2';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://other/volume/path2" in pod "unknown": volume "other" not configured',
      );
    });

    it('handles volume storage without subPath throw volume not mounted', () => {
      const noMountPodTemplateSpec = {
        spec: {
          volumes: [
            {
              name: 'artifact',
              persistentVolumeClaim: {
                claimName: 'artifact_pvc',
              },
            },
          ],
        },
      };
      const logdir = 'volume://artifact/path1';
      expect(() =>
        K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, noMountPodTemplateSpec),
      ).toThrowError(
        'Cannot find file "volume://artifact/path1" in pod "unknown": container "" not found',
      );
    });

    it('handles volume storage without volumeMounts throw volume not mounted', () => {
      const noMountPodTemplateSpec = {
        spec: {
          containers: [
            {
              volumeMounts: [
                {
                  name: 'other',
                  mountPath: '/data',
                },
              ],
            },
          ],
          volumes: [
            {
              name: 'artifact',
              persistentVolumeClaim: {
                claimName: 'artifact_pvc',
              },
            },
          ],
        },
      };
      const logdir = 'volume://artifact/path';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://artifact/path" in pod "unknown": volume "artifact" not mounted',
      );
    });

    it('handles volume storage with subPath throw volume mount not found', () => {
      const logdir = 'volume://artifact/other';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://artifact/other" in pod "unknown": volume "artifact" not mounted or volume "artifact" with subPath (which is prefix of other) not mounted',
      );
    });
  });
});
