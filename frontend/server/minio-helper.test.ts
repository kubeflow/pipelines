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
import { Client as MinioClient } from 'minio';
import { awsInstanceProfileCredentials } from './aws-helper';
import { createMinioClient } from './minio-helper';

jest.mock('minio');
jest.mock('./aws-helper');
const MockedMinioClient: jest.Mock = MinioClient as any;

describe('createMinioClient', () => {
  beforeEach(() => {
    (MinioClient as any).mockClear();
    (awsInstanceProfileCredentials.getCredentials as jest.Mock).mockClear();
    (awsInstanceProfileCredentials.ok as jest.Mock).mockClear();
  });

  it('should create a minio client with the provided configs.', async () => {
    const client = await createMinioClient({
      endPoint: 'minio.kubeflow:80',
      accessKey: 'accesskey',
      secretKey: 'secretkey',
    });

    expect(client).toBeInstanceOf(MinioClient);
    expect(MockedMinioClient).toHaveBeenCalledWith({
      endPoint: 'minio.kubeflow:80',
      accessKey: 'accesskey',
      secretKey: 'secretkey',
    });
  });

  it('should fallback to the provided configs if EC2 metadata is not available.', async () => {
    const client = await createMinioClient({
      endPoint: 'minio.kubeflow:80',
    });

    expect(client).toBeInstanceOf(MinioClient);
    expect(MockedMinioClient).toHaveBeenCalledWith({
      endPoint: 'minio.kubeflow:80',
    });
  });

  it('should use EC2 metadata credentials if access key are not provided.', async () => {
    (awsInstanceProfileCredentials.getCredentials as jest.Mock).mockImplementation(() =>
      Promise.resolve({
        Code: 'Success',
        LastUpdated: '2019-12-17T10:55:38Z',
        Type: 'AWS-HMAC',
        AccessKeyId: 'AccessKeyId',
        SecretAccessKey: 'SecretAccessKey',
        Token: 'SessionToken',
        Expiration: new Date(Date.now() + 1000).toISOString(), // expires 1 sec later
      }),
    );
    (awsInstanceProfileCredentials.ok as jest.Mock).mockImplementation(() => Promise.resolve(true));

    const client = await createMinioClient({ endPoint: 's3.awsamazon.com' });

    expect(client).toBeInstanceOf(MinioClient);
    expect(MockedMinioClient).toHaveBeenCalledWith({
      accessKey: 'AccessKeyId',
      endPoint: 's3.awsamazon.com',
      secretKey: 'SecretAccessKey',
      sessionToken: 'SessionToken',
    });
  });
});
