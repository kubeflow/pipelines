/*
 * Copyright 2018-2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { isS3Endpoint } from './AwsHelper';

describe('isS3Endpoint', () => {
  it('checks a valid s3 endpoint', () => {
    expect(isS3Endpoint('s3.amazonaws.com')).toBe(true);
  });

  it('checks a valid s3 regional endpoint', () => {
    expect(isS3Endpoint('s3.dualstack.us-east-1.amazonaws.com')).toBe(true);
  });

  it('checks a valid s3 cn endpoint', () => {
    expect(isS3Endpoint('s3.cn-north-1.amazonaws.com.cn')).toBe(true);
  });

  it('checks an invalid s3 endpoint', () => {
    expect(isS3Endpoint('amazonaws.com')).toBe(false);
  });

  it('checks non-s3 endpoint', () => {
    expect(isS3Endpoint('minio.kubeflow')).toBe(false);
  });
});
