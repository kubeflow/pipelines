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

jest.mock('node-fetch');

function sleep(ms) {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}

it('getIAMInstanceProfile', async () => {
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
  fetch.mockImplementation(mockFetch);
  sleep;
  expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[0]);
  expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[0]);
  await sleep(1500)
  expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[1]);
  expect(await awsInstanceProfileCredentials.getCredentials()).toBe(expectedCredentials[1]);
});
