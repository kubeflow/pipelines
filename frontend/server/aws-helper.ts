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
function awsEndpointServiceName(endpoint: string): string | undefined {
  let hostname: string;
  try {
    hostname = new URL(endpoint.includes('://') ? endpoint : `https://${endpoint}`).hostname;
  } catch {
    return undefined;
  }

  const normalized = hostname.toLowerCase().replace(/\.$/, '');
  const suffix = normalized.endsWith('.amazonaws.com.cn')
    ? '.amazonaws.com.cn'
    : normalized.endsWith('.amazonaws.com')
      ? '.amazonaws.com'
      : '';
  if (!suffix) {
    return undefined;
  }

  return normalized.slice(0, -suffix.length);
}

/** Returns true only for AWS-operated regional/global S3 service origins. */
export function isOfficialAWSS3ServiceEndpoint(endpoint: string = ''): boolean {
  const serviceName = awsEndpointServiceName(endpoint);
  if (!serviceName) {
    return false;
  }
  const region = '[a-z]{2}(?:-[a-z0-9]+)+-[0-9]+';
  return new RegExp(
    `^(?:s3|s3[.-]${region}|s3(?:-fips)?\\.dualstack\\.${region}|s3-fips[.-]${region})$`,
  ).test(serviceName);
}

/**
 * Check whether an endpoint is supported by the AWS credential chain.
 *
 * This includes AWS-operated S3 service, bucket, access point, Object Lambda,
 * Outposts, and PrivateLink hostnames across commercial, GovCloud, and China
 * partitions.
 */
export function isAWSS3Endpoint(endpoint: string = ''): boolean {
  const serviceName = awsEndpointServiceName(endpoint);
  if (!serviceName) {
    return false;
  }

  const region = '[a-z]{2}(?:-[a-z0-9]+)+-[0-9]+';
  const bucket = '[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?';
  const publicEndpoint = new RegExp(
    `^(?:${bucket}\\.)?(?:s3|s3[.-]${region}|s3(?:-fips)?\\.dualstack\\.${region}|s3-fips[.-]${region})$`,
  );
  const specializedPublicEndpoint = new RegExp(
    `^(?:${bucket}\\.)?(?:s3-accelerate(?:\\.dualstack)?|s3-external-1|s3-accesspoint(?:\\.dualstack)?\\.${region}|s3-object-lambda\\.${region}|s3-outposts\\.${region})$`,
  );
  const privateLinkEndpoint = new RegExp(
    `^(?:${bucket}\\.)?vpce-[a-z0-9-]+\\.s3(?:-accesspoint)?\\.${region}\\.vpce$`,
  );
  return (
    publicEndpoint.test(serviceName) ||
    specializedPublicEndpoint.test(serviceName) ||
    privateLinkEndpoint.test(serviceName)
  );
}
