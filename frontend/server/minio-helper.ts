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
import {Stream} from 'stream';
import * as tar from 'tar';
import {Client as MinioClient, ClientOptions as MinioClientOptions} from 'minio';
import {awsInstanceProfileCredentials} from './aws-helper';


/** IMinioRequestConfig describes the info required to retrieve an artifact. */
export interface IMinioRequestConfig {
    bucket: string;
    key: string;
    client: MinioClient;
}

/** IMinioClientOptionsWithOptionalSecrets wraps around MinioClientOptions where only endPoint is required (accesskey and secretkey are optional). */
export interface IMinioClientOptionsWithOptionalSecrets extends Partial<MinioClientOptions> {
  endPoint: string;
}

/**
 * Create minio client with aws instance profile credentials if needed.
 * @param config minio client options where `accessKey` and `secretKey` are optional.
 */
export async function createMinioClient(config: IMinioClientOptionsWithOptionalSecrets) {

    if (!config.accessKey || !config.secretKey) {
        if (await awsInstanceProfileCredentials.ok()) {
            const credentials = await awsInstanceProfileCredentials.getCredentials();
            if (credentials) {
              const {AccessKeyId: accessKey, SecretAccessKey: secretKey, Token: sessionToken} = credentials;
              return new MinioClient({...config, accessKey, secretKey, sessionToken});
            }
            console.error('unable to get credentials from AWS metadata store.')
        }
    }
    
    return new MinioClient(config as MinioClientOptions);
}

export function getTarObjectAsString({bucket, key, client}: IMinioRequestConfig) {
    return new Promise<string>(async (resolve, reject) => {
      try {
        const stream = await getObjectStream({bucket, key, client});
        let contents = '';
        stream.pipe(new tar.Parse()).on('entry', (entry: Stream) => {
          entry.on('data', (buffer) => contents += buffer.toString());
        });
        stream.on('end', () => {
          resolve(contents);
        });
      } catch (err) {
        reject(err);
      }
    });
}

export function getObjectStream({bucket, key, client}: IMinioRequestConfig) {
  return client.getObject(bucket, key);
}