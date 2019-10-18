/*
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import MinioArtifactLink from './MinioArtifactLink';

describe('MinioArtifactLink', () => {

  it('handles undefined artifact', () => {
    expect(MinioArtifactLink(undefined as any)).toMatchSnapshot();
  });

  it('handles null artifact', () => {
        expect(MinioArtifactLink(null as any)).toMatchSnapshot();
  });

  it('handles empty artifact', () => {
    expect(MinioArtifactLink({} as any)).toMatchSnapshot();
  });

  it('handles invalid artifact: no bucket', () => {
    const s3artifact = {
      accessKeySecret: {key: 'accesskey', optional: false, name: 'minio'},
      bucket: '',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: {key: 'secretkey', optional: false, name: 'minio'},
    };
    expect(MinioArtifactLink(s3artifact)).toMatchSnapshot();
  });

  it('handles invalid artifact: no key', () => {
    const s3artifact = {
      accessKeySecret: {key: 'accesskey', optional: false, name: 'minio'},
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: '',
      secretKeySecret: {key: 'secretkey', optional: false, name: 'minio'},
    };
    expect(MinioArtifactLink(s3artifact)).toMatchSnapshot();
  });

  it('handles s3 artifact', () => {
    const s3artifact = {
      accessKeySecret: {key: 'accesskey', optional: false, name: 'minio'},
      bucket: 'foo',
      endpoint: 's3.amazonaws.com',
      key: 'bar',
      secretKeySecret: {key: 'secretkey', optional: false, name: 'minio'},
    };
    expect(MinioArtifactLink(s3artifact)).toMatchSnapshot();
  });

  it('handles minio artifact', () => {
    const minioartifact = {
      accessKeySecret: {key: 'accesskey', optional: false, name: 'minio'},
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: {key: 'secretkey', optional: false, name: 'minio'},
    };
    expect(MinioArtifactLink(minioartifact)).toMatchSnapshot();
  });

});