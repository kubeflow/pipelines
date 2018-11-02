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
  const spy = jest.fn(() => Promise.resolve({
    ok: true,
    text: () => response,
  }));
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

  it('getPodLogs', async () => {
    const spy = fetchSpy('http://some/address');
    expect(await Apis.getPodLogs('some-pod-name')).toEqual('http://some/address');
    expect(spy).toHaveBeenCalledWith(
      'k8s/pod/logs?podname=some-pod-name', { credentials: 'same-origin' });
  });

  it('getPodLogs error', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => null);
    window.fetch = jest.fn(() => Promise.resolve({
      ok: false,
      text: () => 'bad response',
    }));
    expect(Apis.getPodLogs('some-pod-name')).rejects.toThrowError('bad response');
  });

  it('isApiServerReady returns true', async () => {
    fetchSpy(JSON.stringify({ apiServerReady: true }));
    expect(await Apis.isApiServerReady()).toEqual(true);
  });

  it('isApiServerReady returns false on unreachable', async () => {
    window.fetch = () => { throw new Error('nope!'); };
    expect(await Apis.isApiServerReady()).toEqual(false);
  });

  it('isApiServerReady returns false on bad json', async () => {
    fetchSpy('bad json');
    expect(await Apis.isApiServerReady()).toEqual(false);
  });

  it('readFile', async () => {
    const spy = fetchSpy('file contents');
    expect(await Apis.readFile({
      bucket: 'testbucket',
      key: 'testkey',
      source: StorageService.GCS,
    })).toEqual('file contents');
    expect(spy).toHaveBeenCalledWith(
      'artifacts/get?source=gcs&bucket=testbucket&key=testkey', { credentials: 'same-origin' });
  });

  it('getTensorboardApp', async () => {
    const spy = fetchSpy('http://some/address');
    expect(await Apis.getTensorboardApp('gs://log/dir')).toEqual('http://some/address');
    expect(spy).toHaveBeenCalledWith(
      'apps/tensorboard?logdir=' + encodeURIComponent('gs://log/dir'),
      { credentials: 'same-origin' },
    );
  });

  it('startTensorboardApp', async () => {
    const spy = fetchSpy('http://some/address');
    await Apis.startTensorboardApp('gs://log/dir');
    expect(spy).toHaveBeenCalledWith(
      'apps/tensorboard?logdir=' + encodeURIComponent('gs://log/dir'),
      { credentials: 'same-origin', method: 'POST', headers: { 'content-type': 'application/json' } },
    );
  });

  it('uploadPipeline', async () => {
    const spy = fetchSpy(JSON.stringify({ name: 'resultName' }));
    const result = await Apis.uploadPipeline('test pipeline name', new File([], 'test name'));
    expect(result).toEqual({ name: 'resultName' });
    expect(spy).toHaveBeenCalledWith(
      'apis/v1beta1/pipelines/upload?name=' + encodeURIComponent('test pipeline name'),
      {
        body: expect.anything(),
        cache: 'no-cache',
        credentials: 'same-origin',
        method: 'POST',
      }
    );
  });
});
