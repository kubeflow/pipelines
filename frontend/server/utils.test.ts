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
import { PassThrough } from 'stream';
import { PreviewStream, findFileOnPodVolume, resolveFilePathOnVolume } from './utils';

describe('utils', () => {
  describe('PreviewStream', () => {
    it('should stream first 5 bytes', done => {
      const peek = 5;
      const input = 'some string that will be truncated.';
      const source = new PassThrough();
      const preview = new PreviewStream({ peek });
      const dst = source.pipe(preview).on('end', done);
      source.end(input);
      dst.once('readable', () => expect(dst.read().toString()).toBe(input.slice(0, peek)));
    });

    it('should stream everything if peek==0', done => {
      const peek = 0;
      const input = 'some string that will be truncated.';
      const source = new PassThrough();
      const preview = new PreviewStream({ peek });
      const dst = source.pipe(preview).on('end', done);
      source.end(input);
      dst.once('readable', () => expect(dst.read().toString()).toBe(input));
    });
  });

  describe('findFileOnPodVolume', () => {
    const podTemplateSpec = {
      spec: {
        containers: [
          {
            volumeMounts: [
              {
                name: 'output',
                mountPath: '/main',
              },
              {
                name: 'artifact',
                subPath: 'pipeline1',
                mountPath: '/main1',
              },
              {
                name: 'artifact',
                subPath: 'pipeline2',
                mountPath: '/main2',
              },
            ],
            name: 'main',
          },
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
            name: 'ml-pipeline-ui',
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

    it('parse file path with containerNames', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: ['ml-pipeline-ui', 'ml-pipeline-ui-artifact'],
        volumeMountName: 'output',
        filePathInVolume: 'a/b/c',
      });
      expect(err).toEqual(undefined);
      expect(filePath).toEqual('/data/a/b/c');
    });

    it('parse file path with containerNames and subPath', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: ['ml-pipeline-ui', 'ml-pipeline-ui-artifact'],
        volumeMountName: 'artifact',
        filePathInVolume: 'pipeline1/a/b/c',
      });
      expect(err).toEqual(undefined);
      expect(filePath).toEqual('/data1/a/b/c');
    });

    it('parse file path without containerNames', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: undefined,
        volumeMountName: 'output',
        filePathInVolume: 'a/b/c',
      });
      expect(err).toEqual(undefined);
      expect(filePath).toEqual('/main/a/b/c');
    });

    it('parse file path error with not exist volume', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: undefined,
        volumeMountName: 'other',
        filePathInVolume: 'a/b/c',
      });
      expect(err).toEqual(
        'Cannot find file "volume://other/a/b/c" in pod "unknown": volume "other" not configured',
      );
      expect(filePath).toEqual('');
    });

    it('parse file path error with not exist container', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: ['other1', 'other2'],
        volumeMountName: 'output',
        filePathInVolume: 'a/b/c',
      });
      expect(err).toEqual(
        'Cannot find file "volume://output/a/b/c" in pod "unknown": container "other1" or "other2" not found',
      );
      expect(filePath).toEqual('');
    });

    it('parse file path error with volume not mount error', () => {
      const [filePath, err] = findFileOnPodVolume(podTemplateSpec, {
        containerNames: undefined,
        volumeMountName: 'artifact',
        filePathInVolume: 'a/b/c',
      });
      expect(err).toEqual(
        'Cannot find file "volume://artifact/a/b/c" in pod "unknown": volume "artifact" not mounted or volume "artifact" with subPath (which is prefix of a/b/c) not mounted',
      );
      expect(filePath).toEqual('');
    });
  });

  describe('resolveFilePathOnVolume', () => {
    it('undefined volumeMountSubPath', () => {
      const path = resolveFilePathOnVolume({
        filePathInVolume: 'a/b/c',
        volumeMountPath: '/data',
        volumeMountSubPath: undefined,
      });
      expect(path).toEqual(['/data/a/b/c', undefined]);
    });

    it('with volumeMountSubPath', () => {
      const path = resolveFilePathOnVolume({
        volumeMountPath: '/data',
        filePathInVolume: 'a/b/c',
        volumeMountSubPath: 'a',
      });
      expect(path).toEqual(['/data/b/c', undefined]);
    });

    it('with multiple layer volumeMountSubPath', () => {
      const path = resolveFilePathOnVolume({
        volumeMountPath: '/data',
        filePathInVolume: 'a/b/c',
        volumeMountSubPath: 'a/b',
      });
      expect(path).toEqual(['/data/c', undefined]);
    });

    it('with not exist volumeMountSubPath', () => {
      const path = resolveFilePathOnVolume({
        volumeMountPath: '/data',
        filePathInVolume: 'a/b/c',
        volumeMountSubPath: 'other',
      });
      expect(path).toEqual([
        '',
        'File a/b/c not mounted, expecting the file to be inside volume mount subpath other',
      ]);
    });
  });
});
