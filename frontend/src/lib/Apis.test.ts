// Copyright 2018 Google LLC
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

import { Apis } from './Apis';
import { StorageService } from './WorkflowParser';

const fetchSpy = (response: string) => {
  const spy = jest.fn(() =>
    Promise.resolve({
      ok: true,
      text: () => response,
    }),
  );
  window.fetch = spy;
  return spy;
};

const failedFetchSpy = (response: string) => {
  const spy = jest.fn(() =>
    Promise.resolve({
      ok: false,
      text: () => response,
    }),
  );
  window.fetch = spy;
  return spy;
};

describe('Apis', () => {
  it('hosts a singleton experimentServiceApi', () => {
    expect(Apis.experimentServiceApi).toBe(Apis.experimentServiceApi);
  });

  it('hosts a singleton jobServiceApi', () => {
    expect(Apis.jobServiceApi).toBe(Apis.jobServiceApi);
  });

  it('hosts a singleton pipelineServiceApi', () => {
    expect(Apis.pipelineServiceApi).toBe(Apis.pipelineServiceApi);
  });

  it('hosts a singleton runServiceApi', () => {
    expect(Apis.runServiceApi).toBe(Apis.runServiceApi);
  });

  it('hosts a singleton visualizationServiceApi', () => {
    expect(Apis.visualizationServiceApi).toBe(Apis.visualizationServiceApi);
  });

  it('getPodLogs', async () => {
    const spy = fetchSpy('http://some/address');
    expect(await Apis.getPodLogs('some-pod-name')).toEqual('http://some/address');
    expect(spy).toHaveBeenCalledWith('k8s/pod/logs?podname=some-pod-name', {
      credentials: 'same-origin',
    });
  });

  it('getPodLogs in a specific namespace', async () => {
    const spy = fetchSpy('http://some/address');
    expect(await Apis.getPodLogs('some-pod-name', 'some-namespace-name')).toEqual(
      'http://some/address',
    );
    expect(spy).toHaveBeenCalledWith(
      'k8s/pod/logs?podname=some-pod-name&podnamespace=some-namespace-name',
      {
        credentials: 'same-origin',
      },
    );
  });

  it('getPodLogs error', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => null);
    window.fetch = jest.fn(() =>
      Promise.resolve({
        ok: false,
        text: () => 'bad response',
      }),
    );
    expect(Apis.getPodLogs('some-pod-name')).rejects.toThrowError('bad response');
    expect(Apis.getPodLogs('some-pod-name', 'some-namespace-name')).rejects.toThrowError(
      'bad response',
    );
  });

  it('getBuildInfo returns build information', async () => {
    const expectedBuildInfo = {
      apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      apiServerReady: true,
      buildDate: 'Wed Jan 9 19:40:24 UTC 2019',
      frontendCommitHash: '8efb2fcff9f666ba5b101647e909dc9c6889cecb',
    };
    const spy = fetchSpy(JSON.stringify(expectedBuildInfo));
    const actualBuildInfo = await Apis.getBuildInfo();
    expect(spy).toHaveBeenCalledWith('apis/v1beta1/healthz', { credentials: 'same-origin' });
    expect(actualBuildInfo).toEqual(expectedBuildInfo);
  });

  it('isJupyterHubAvailable returns true if the response for the /hub/ url was ok', async () => {
    const spy = fetchSpy('{}');
    const isJupyterHubAvailable = await Apis.isJupyterHubAvailable();
    expect(spy).toHaveBeenCalledWith('/hub/', { credentials: 'same-origin' });
    expect(isJupyterHubAvailable).toEqual(true);
  });

  it('isJupyterHubAvailable returns false if the response for the /hub/ url was not ok', async () => {
    const spy = jest.fn(() => Promise.resolve({ ok: false }));
    window.fetch = spy;
    const isJupyterHubAvailable = await Apis.isJupyterHubAvailable();
    expect(spy).toHaveBeenCalledWith('/hub/', { credentials: 'same-origin' });
    expect(isJupyterHubAvailable).toEqual(false);
  });

  it('readFile', async () => {
    const spy = fetchSpy('file contents');
    expect(
      await Apis.readFile({
        bucket: 'testbucket',
        key: 'testkey',
        source: StorageService.GCS,
      }),
    ).toEqual('file contents');
    expect(spy).toHaveBeenCalledWith('artifacts/get?source=gcs&bucket=testbucket&key=testkey', {
      credentials: 'same-origin',
    });
  });

  it('buildReadFileUrl', () => {
    expect(
      Apis.buildReadFileUrl(
        {
          bucket: 'testbucket',
          key: 'testkey',
          source: StorageService.GCS,
        },
        'testnamespace',
        255,
      ),
    ).toEqual(
      'artifacts/get?source=gcs&namespace=testnamespace&peek=255&bucket=testbucket&key=testkey',
    );
  });

  it('buildArtifactUrl', () => {
    expect(
      Apis.buildArtifactUrl({
        bucket: 'testbucket',
        key: 'testkey',
        source: StorageService.GCS,
      }),
    ).toEqual('gcs://testbucket/testkey');
  });

  it('getTensorboardApp', async () => {
    const spy = fetchSpy(
      JSON.stringify({ podAddress: 'http://some/address', tfVersion: '1.14.0' }),
    );
    const tensorboardInstance = await Apis.getTensorboardApp('gs://log/dir', 'test-ns');
    expect(tensorboardInstance).toEqual({ podAddress: 'http://some/address', tfVersion: '1.14.0' });
    expect(spy).toHaveBeenCalledWith(
      `apps/tensorboard?logdir=${encodeURIComponent('gs://log/dir')}&namespace=test-ns`,
      { credentials: 'same-origin' },
    );
  });

  it('startTensorboardApp', async () => {
    const spy = fetchSpy('http://some/address');
    await Apis.startTensorboardApp('gs://log/dir', '1.14.0', 'test-ns');
    expect(spy).toHaveBeenCalledWith(
      'apps/tensorboard?logdir=' +
        encodeURIComponent('gs://log/dir') +
        '&tfversion=1.14.0' +
        '&namespace=test-ns',
      {
        credentials: 'same-origin',
        headers: { 'content-type': 'application/json' },
        method: 'POST',
      },
    );
  });

  it('deleteTensorboardApp', async () => {
    const spy = fetchSpy('http://some/address');
    await Apis.deleteTensorboardApp('gs://log/dir', 'test-ns');
    expect(spy).toHaveBeenCalledWith(
      'apps/tensorboard?logdir=' + encodeURIComponent('gs://log/dir') + '&namespace=test-ns',
      {
        credentials: 'same-origin',
        method: 'DELETE',
      },
    );
  });

  it('uploadPipeline', async () => {
    const spy = fetchSpy(JSON.stringify({ name: 'resultName' }));
    const result = await Apis.uploadPipeline(
      'test pipeline name',
      'test description',
      new File([], 'test name'),
    );
    expect(result).toEqual({ name: 'resultName' });
    expect(spy).toHaveBeenCalledWith(
      'apis/v1beta1/pipelines/upload?name=' +
        encodeURIComponent('test pipeline name') +
        '&description=' +
        encodeURIComponent('test description'),
      {
        body: expect.anything(),
        cache: 'no-cache',
        credentials: 'same-origin',
        method: 'POST',
      },
    );
  });

  it('checks if Tensorboard pod is ready', async () => {
    const spy = fetchSpy('');
    const ready = await Apis.isTensorboardPodReady('apis/v1beta1/_proxy/pod_address');
    expect(ready).toBe(true);
    expect(spy).toHaveBeenCalledWith('apis/v1beta1/_proxy/pod_address', { method: 'HEAD' });
  });

  it('checks if Tensorboard pod is not ready', async () => {
    const spy = failedFetchSpy('');
    const ready = await Apis.isTensorboardPodReady('apis/v1beta1/_proxy/pod_address');
    expect(ready).toBe(false);
    expect(spy).toHaveBeenCalledWith('apis/v1beta1/_proxy/pod_address', { method: 'HEAD' });
  });
});
