/*
 * Copyright 2019-2020 Google LLC
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
import React from 'react';
import { render, act } from '@testing-library/react';
import MinioArtifactPreview from './MinioArtifactPreview';
import { Apis } from '../lib/Apis';
import TestUtils from '../TestUtils';

jest.mock('../lib/Apis');

describe('MinioArtifactPreview', () => {
  const readFile = jest.spyOn(Apis, 'readFile');

  beforeEach(() => {
    readFile.mockResolvedValue('preview ...');
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it('handles undefined artifact', () => {
    const { container } = render(<MinioArtifactPreview value={undefined} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles null artifact', () => {
    const { container } = render(<MinioArtifactPreview value={null as any} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles empty artifact', () => {
    const { container } = render(<MinioArtifactPreview value={{} as any} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles invalid artifact: no bucket', () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: '',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactPreview value={s3Artifact} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles invalid artifact: no key', () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: '',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(<MinioArtifactPreview value={s3Artifact} />);
    expect(container).toMatchInlineSnapshot(`<div />`);
  });

  it('handles string value', () => {
    const { container } = render(<MinioArtifactPreview value='teststring' />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        teststring
      </div>
    `);
  });

  it('handles boolean value', () => {
    const { container } = render(<MinioArtifactPreview value={false as any} />);
    expect(container).toMatchInlineSnapshot(`
      <div>
        false
      </div>
    `);
  });

  it('handles s3 artifact', async () => {
    const s3Artifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 's3.amazonaws.com',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };

    const { container } = render(<MinioArtifactPreview value={s3Artifact} />);
    await act(TestUtils.flushPromises);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="root"
        >
          <a
            class="link"
            rel="noopener"
            target="_blank"
          />
          <div
            class="preview"
          >
            <small>
              <pre>
                preview ...
              </pre>
            </small>
          </div>
        </div>
      </div>
    `);
  });

  it('handles minio artifact', async () => {
    const minioArtifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const container = document.body.appendChild(document.createElement('div'));
    await act(async () => {
      render(<MinioArtifactPreview value={minioArtifact} />, { container });
    });
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="root"
        >
          <a
            class="link"
            rel="noopener"
            target="_blank"
          />
          <div
            class="preview"
          >
            <small>
              <pre>
                preview ...
              </pre>
            </small>
          </div>
        </div>
      </div>
    `);
  });

  it('handles artifact with namespace', async () => {
    const minioArtifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    const { container } = render(
      <MinioArtifactPreview value={minioArtifact} namespace='namespace' />,
    );
    await act(TestUtils.flushPromises);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="root"
        >
          <a
            class="link"
            rel="noopener"
            target="_blank"
          />
          <div
            class="preview"
          >
            <small>
              <pre>
                preview ...
              </pre>
            </small>
          </div>
        </div>
      </div>
    `);
  });

  it('handles artifact cleanly even when fetch fails', async () => {
    const minioArtifact = {
      accessKeySecret: { key: 'accesskey', optional: false, name: 'minio' },
      bucket: 'foo',
      endpoint: 'minio.kubeflow',
      key: 'bar',
      secretKeySecret: { key: 'secretkey', optional: false, name: 'minio' },
    };
    readFile.mockRejectedValue('unknown error');
    const { container } = render(<MinioArtifactPreview value={minioArtifact} />);
    await act(TestUtils.flushPromises);
    expect(container).toMatchInlineSnapshot(`
      <div>
        <div
          class="root"
        >
          <a
            class="link"
            rel="noopener"
            target="_blank"
          />
        </div>
      </div>
    `);
  });
});
