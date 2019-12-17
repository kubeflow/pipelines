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
import fetch from 'node-fetch';
import { awsInstanceProfileCredentials } from './aws-helper';

// mock node-fetch module
jest.mock('node-fetch');

function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

describe('awsInstanceProfileCredentials', () => {
  const mockedFetch: jest.Mock = fetch as any;

  afterEach(() => {
    awsInstanceProfileCredentials.reset();
    mockedFetch.mockReset();
  });

  it('should retrieve, cache, and refresh AWS EC2 instance profile and session credentials', async () => {
    let count = 0;
    const expectedCredentials = [
      {
        Code: 'Success',
        LastUpdated: '2019-12-17T10:55:38Z',
        Type: 'AWS-HMAC',
        AccessKeyId: 'AccessKeyId',
        SecretAccessKey: 'SecretAccessKey',
        Token: 'SessionToken',
        Expiration: new Date(Date.now() + 1000).toISOString(), // expires 1 sec later
      },
      {
        Code: 'Success',
        LastUpdated: '2019-12-17T10:55:38Z',
        Type: 'AWS-HMAC',
        AccessKeyId: 'AccessKeyId2',
        SecretAccessKey: 'SecretAccessKey2',
        Token: 'SessionToken2',
        Expiration: new Date(Date.now() + 10000).toISOString(), // expires 10 sec later
      },
    ];
    const mockFetch = (url: string) => {
      if (url === 'http://169.254.169.254/latest/meta-data/iam/security-credentials/') {
        return Promise.resolve({ text: () => Promise.resolve('some_iam_role') });
      }
      return Promise.resolve({
        json: () => Promise.resolve(expectedCredentials[count++]),
      });
    };
    mockedFetch.mockImplementation(mockFetch);

    // expect to get cred from ec2 instance metadata store
    expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[0]);
    // expect to call once for profile name, and once for credential
    expect(mockedFetch.mock.calls.length).toBe(2);
    // expect to get same credential as it has not expire
    expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[0]);
    // expect to not to have any more calls
    expect(mockedFetch.mock.calls.length).toBe(2);
    // let credential expire
    await sleep(1500);
    // expect to get new cred as old one expire
    expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[1]);
    // expect to get same cred as it has not expire
    expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[1]);
  });

  it('should fail gracefully if there is no instance profile.', async () => {
    const mockFetch = (url: string) => {
      if (url === 'http://169.254.169.254/latest/meta-data/iam/security-credentials/') {
        return Promise.resolve({ text: () => Promise.resolve('') });
      }
      return Promise.reject('Unknown error');
    };
    mockedFetch.mockImplementation(mockFetch);

    expect(await awsInstanceProfileCredentials.ok()).toBeFalsy();
    expect(awsInstanceProfileCredentials.getCredentials).not.toThrow();
    expect(await awsInstanceProfileCredentials.getCredentials()).toBeUndefined();
  });

  it('should fail gracefully if there is no metadata store.', async () => {
    const mockFetch = (_: string) => {
      return Promise.reject('Unknown error');
    };
    mockedFetch.mockImplementation(mockFetch);

    expect(await awsInstanceProfileCredentials.ok()).toBeFalsy();
    expect(awsInstanceProfileCredentials.getCredentials).not.toThrow();
    expect(await awsInstanceProfileCredentials.getCredentials()).toBeUndefined();
  });
});
