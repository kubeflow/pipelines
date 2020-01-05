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
import { PassThrough } from 'stream';
import { Client as MinioClient } from 'minio';
import { awsInstanceProfileCredentials } from './aws-helper';
import { createMinioClient, getTarObjectAsString } from './minio-helper';

jest.mock('minio');
jest.mock('./aws-helper');

describe('minio-helper', () => {
  const MockedMinioClient: jest.Mock = MinioClient as any;

  beforeEach(() => {
    jest.resetAllMocks();
  });

  describe('createMinioClient', () => {
    it('creates a minio client with the provided configs.', async () => {
      const client = await createMinioClient({
        accessKey: 'accesskey',
        endPoint: 'minio.kubeflow:80',
        secretKey: 'secretkey',
      });

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'minio.kubeflow:80',
        secretKey: 'secretkey',
      });
    });

    it('fallbacks to the provided configs if EC2 metadata is not available.', async () => {
      const client = await createMinioClient({
        endPoint: 'minio.kubeflow:80',
      });

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        endPoint: 'minio.kubeflow:80',
      });
    });

    it('uses EC2 metadata credentials if access key are not provided.', async () => {
      (awsInstanceProfileCredentials.getCredentials as jest.Mock).mockImplementation(() =>
        Promise.resolve({
          AccessKeyId: 'AccessKeyId',
          Code: 'Success',
          Expiration: new Date(Date.now() + 1000).toISOString(), // expires 1 sec later
          LastUpdated: '2019-12-17T10:55:38Z',
          SecretAccessKey: 'SecretAccessKey',
          Token: 'SessionToken',
          Type: 'AWS-HMAC',
        }),
      );
      (awsInstanceProfileCredentials.ok as jest.Mock).mockImplementation(() =>
        Promise.resolve(true),
      );

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

  describe('getTarObjectAsString', () => {
    // hello world
    const tarGzBase64 =
      'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
    const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');

    it('unpacks the tar gz and return the string "hello world".', async () => {
      const client = new MinioClient({
        accessKey: 'minio',
        endPoint: 'minio-service.kubeflow',
        secretKey: 'minio123',
        useSSL: false,
      });
      const mockedGetObject: jest.Mock = client.getObject as any;
      const stream = new PassThrough();
      stream.end(tarGzBuffer);
      mockedGetObject.mockResolvedValue(stream);

      const content = await getTarObjectAsString({
        bucket: 'bucket',
        client,
        key: 'key',
      });

      expect(content).toBe('hello world\n');
    });
  });
});
