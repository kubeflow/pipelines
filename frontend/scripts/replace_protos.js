/*
 * Copyright 2021 The Kubeflow Authors
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


const fs = require('fs');
const request = require('request');

const default_mlmd_version = "0.25.1";
const mlmd_version = process.argv.slice(2)[0] || default_mlmd_version;

// Step#1 delete two original files
const metadata_store_file_path = "../third_party/ml-metadata/ml_metadata/proto/metadata_store.proto";
const metadata_store_service_file_path = "../third_party/ml-metadata/ml_metadata/proto/metadata_store_service.proto";

try {
  if (fs.existsSync(metadata_store_file_path)) {
    fs.unlinkSync(metadata_store_file_path);
  }

  if (fs.existsSync(metadata_store_service_file_path)) {
    fs.unlinkSync(metadata_store_service_file_path);
  }
  console.log("Step#1 finished: deleted two original files!");
} catch(err) {
  console.error(err)
}

// Step#2 download two new proto files
const new_metadata_store_file_path = "https://raw.githubusercontent.com/google/ml-metadata/v" + mlmd_version + "/ml_metadata/proto/metadata_store.proto";
const new_metadata_store_service_file_path = "https://raw.githubusercontent.com/google/ml-metadata/v" + mlmd_version + "/ml_metadata/proto/metadata_store_service.proto";
download(
    new_metadata_store_file_path,
    metadata_store_file_path,
    console.log
);
download(
    new_metadata_store_service_file_path,
    metadata_store_service_file_path,
    console.log
);
console.log("Step#2 finished: downloaded two new files!");

function download (url, dest, cb) {
  const file = fs.createWriteStream(dest);
  const sendReq = request.get(url);

  // verify response code
  sendReq.on('response', (response) => {
    if (response.statusCode !== 200) {
      return cb('Response status was ' + response.statusCode);
    }

    console.log("File " + dest + " Downloaded!");

    sendReq.pipe(file);
  });

  // close() is async, call cb after close completes
  file.on('finish', () => file.close(cb));

  // check for request errors
  sendReq.on('error', (err) => {
    fs.unlink(dest);
    return cb(err.message);
  });

  file.on('error', (err) => { // Handle errors
    fs.unlink(dest); // Delete the file async. (But we don't check the result)
    return cb(err.message);
  });
}
