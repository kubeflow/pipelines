// Copyright 2019 The Kubeflow Authors
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
import { describe, it, expect } from 'vitest';
import { isAWSS3Endpoint, isOfficialAWSS3ServiceEndpoint } from './aws-helper.js';

describe('isS3Endpoint', () => {
  it('checks a valid s3 endpoint', () => {
    expect(isAWSS3Endpoint('s3.amazonaws.com')).toBe(true);
  });

  it('checks a valid s3 regional endpoint', () => {
    expect(isAWSS3Endpoint('s3.dualstack.us-east-1.amazonaws.com')).toBe(true);
  });

  it('checks a valid s3 cn endpoint', () => {
    expect(isAWSS3Endpoint('s3.cn-north-1.amazonaws.com.cn')).toBe(true);
  });

  it.each([
    's3-fips.us-gov-west-1.amazonaws.com',
    's3-fips.us-gov-east-1.amazonaws.com',
    's3-fips-us-gov-west-1.amazonaws.com',
    's3-fips-us-gov-east-1.amazonaws.com',
  ])('recognizes the GovCloud FIPS service endpoint %s', (endpoint) => {
    expect(isAWSS3Endpoint(endpoint)).toBe(true);
    expect(isOfficialAWSS3ServiceEndpoint(endpoint)).toBe(true);
  });

  it.each([
    'tenant-bucket.s3-fips-us-gov-west-1.amazonaws.com',
    'tenant-bucket.s3-fips-us-gov-east-1.amazonaws.com',
  ])('distinguishes the GovCloud FIPS bucket endpoint %s from a service origin', (endpoint) => {
    expect(isAWSS3Endpoint(endpoint)).toBe(true);
    expect(isOfficialAWSS3ServiceEndpoint(endpoint)).toBe(false);
  });

  it.each([
    's3-fips-us-gov-west-1.amazonaws.com.attacker.test',
    's3-fips-us-gov-west-1.attacker.amazonaws.com',
    's3-fips-us-gov-west-1.elb.amazonaws.com',
  ])('rejects a hostname impersonating a GovCloud FIPS endpoint: %s', (endpoint) => {
    expect(isAWSS3Endpoint(endpoint)).toBe(false);
    expect(isOfficialAWSS3ServiceEndpoint(endpoint)).toBe(false);
  });

  it('checks a valid s3 PrivateLink endpoint', () => {
    expect(isAWSS3Endpoint('vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com')).toBe(true);
  });

  it.each([
    'tenant-bucket.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com',
    'access-point-123456789012.vpce-1a2b3c4d-5e6f.s3-accesspoint.us-east-1.vpce.amazonaws.com',
  ])('checks a bucket or access-point S3 PrivateLink endpoint %s', (endpoint) => {
    expect(isAWSS3Endpoint(endpoint)).toBe(true);
  });

  it.each([
    's3-accelerate.amazonaws.com',
    'tenant-bucket.s3-accelerate.dualstack.amazonaws.com',
    'access-point-123456789012.s3-accesspoint.us-east-1.amazonaws.com',
    'access-point-123456789012.s3-accesspoint.dualstack.us-east-1.amazonaws.com',
    'access-point-123456789012.s3-object-lambda.us-east-1.amazonaws.com',
    'access-point-123456789012.op-0123456789abcdef0.s3-outposts.us-east-1.amazonaws.com',
    's3-external-1.amazonaws.com',
  ])('recognizes supported AWS credential-chain endpoint %s', (endpoint) => {
    expect(isAWSS3Endpoint(endpoint)).toBe(true);
  });

  it('checks an invalid s3 endpoint', () => {
    expect(isAWSS3Endpoint('amazonaws.com')).toBe(false);
  });

  it('rejects a non-S3 AWS service hostname containing an s3-like label', () => {
    expect(isAWSS3Endpoint('s3-attacker-123.us-east-1.elb.amazonaws.com')).toBe(false);
    expect(isAWSS3Endpoint('tenant.s3-accesspoint.attacker.amazonaws.com')).toBe(false);
  });

  it('checks non-s3 endpoint', () => {
    expect(isAWSS3Endpoint('minio.kubeflow')).toBe(false);
  });

  it('distinguishes AWS S3 service endpoints from bucket endpoints', () => {
    expect(isOfficialAWSS3ServiceEndpoint('s3.us-east-1.amazonaws.com')).toBe(true);
    expect(isOfficialAWSS3ServiceEndpoint('tenant-bucket.s3.us-east-1.amazonaws.com')).toBe(false);
  });
});
