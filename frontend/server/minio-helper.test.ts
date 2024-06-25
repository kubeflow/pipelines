// Copyright 2019-2020 The Kubeflow Authors
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
import * as zlib from 'zlib';
import { PassThrough } from 'stream';
import { Client as MinioClient } from 'minio';
import { createMinioClient, isTarball, maybeTarball, getObjectStream } from './minio-helper';
const { fromNodeProviderChain } = require('@aws-sdk/credential-providers');

jest.mock('minio');
jest.mock('@aws-sdk/credential-providers');

describe('minio-helper', () => {
  const MockedMinioClient: jest.Mock = MinioClient as any;
  const MockedAuthorizeFn: jest.Mock = jest.fn(x => undefined);

  beforeEach(() => {
    jest.resetAllMocks();
  });

  describe('createMinioClient', () => {
    it('creates a minio client with the provided configs.', async () => {
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'minio.kubeflow:80',
          secretKey: 'secretkey',
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'minio.kubeflow:80',
        secretKey: 'secretkey',
      });
    });

    it('fallbacks to the provided configs if EC2 metadata is not available.', async () => {
      const client = await createMinioClient(
        {
          endPoint: 'minio.kubeflow:80',
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        endPoint: 'minio.kubeflow:80',
      });
    });

    it('uses EC2 metadata credentials if access key are not provided.', async () => {
      (fromNodeProviderChain as jest.Mock).mockImplementation(() => () =>
        Promise.resolve({
          accessKeyId: 'AccessKeyId',
          secretAccessKey: 'SecretAccessKey',
          sessionToken: 'SessionToken',
        }),
      );
      const client = await createMinioClient({ endPoint: 's3.amazonaws.com' }, 's3');
      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'AccessKeyId',
        endPoint: 's3.amazonaws.com',
        secretKey: 'SecretAccessKey',
        sessionToken: 'SessionToken',
      });
      expect(MockedMinioClient).toBeCalledTimes(1);
    });
  });

  describe('isTarball', () => {
    it('checks magic number in buffer is a tarball.', () => {
      const tarGzBase64 =
        'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
      const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
      const tarBuffer = zlib.gunzipSync(tarGzBuffer);

      expect(isTarball(tarBuffer)).toBe(true);
    });

    it('checks magic number in buffer is not a tarball.', () => {
      expect(
        isTarball(
          Buffer.from(
            'some-random-string-more-random-string-even-more-random-string-even-even-more-random',
          ),
        ),
      ).toBe(false);
    });
  });

  describe('maybeTarball', () => {
    // hello world
    const tarGzBase64 =
      'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
    const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
    const tarBuffer = zlib.gunzipSync(tarGzBuffer);

    it('return the content for the 1st file inside a tarball', done => {
      const stream = new PassThrough();
      const maybeTar = stream.pipe(maybeTarball());
      stream.end(tarBuffer);
      stream.on('end', () => {
        expect(maybeTar.read().toString()).toBe('hello world\n');
        done();
      });
    });

    it('return the content normal if is not a tarball', done => {
      const stream = new PassThrough();
      const maybeTar = stream.pipe(maybeTarball());
      stream.end('hello world');
      stream.on('end', () => {
        expect(maybeTar.read().toString()).toBe('hello world');
        done();
      });
    });
  });

  describe('getObjectStream', () => {
    // hello world
    const tarGzBase64 =
      'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
    const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
    const tarBuffer = zlib.gunzipSync(tarGzBuffer);
    let minioClient: MinioClient;
    let mockedMinioGetObject: jest.Mock;

    beforeEach(() => {
      jest.clearAllMocks();
      minioClient = new MinioClient({
        endPoint: 's3.amazonaws.com',
        accessKey: '',
        secretKey: '',
      });
      mockedMinioGetObject = minioClient.getObject as any;
    });

    it('unpacks a gzipped tarball', async done => {
      const objStream = new PassThrough();
      objStream.end(tarGzBuffer);
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(
          stream
            .read()
            .toString()
            .trim(),
        ).toBe('hello world');
        done();
      });
    });

    it('unpacks a uncompressed tarball', async done => {
      const objStream = new PassThrough();
      objStream.end(tarBuffer);
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(
          stream
            .read()
            .toString()
            .trim(),
        ).toBe('hello world');
        done();
      });
    });

    it('returns the content as a stream', async done => {
      const objStream = new PassThrough();
      objStream.end('hello world');
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(
          stream
            .read()
            .toString()
            .trim(),
        ).toBe('hello world');
        done();
      });
    });
  });
});
