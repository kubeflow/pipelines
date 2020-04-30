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

/**
 * Check if the provided string is an S3 endpoint (can be any region).
 *
 * @param endpoint minio endpoint to check.
 */
export function isS3Endpoint(endpoint: string = ''): boolean {
  return !!endpoint.match(/s3.{0,}\.amazonaws\.com\.?.{0,}/i);
}
