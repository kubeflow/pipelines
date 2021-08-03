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

// Note: This script uses protobuf.js to generate pipeline_spec classes in Typescript.
// Reason: protoc doesn't decode from plain json payload to proto object.
//         protobuf.js supports this feature along with other protobuf functionality.
// Usage: Under frontend/, run: `npm run build:pipeline-spec`.
// Output: Result is stored in frontend/src/third_party/pipeline_spec.

// Explaination of protobufjs cli tool:
// Install protobufjs-cli by using the main library
// Command: npm install protobufjs --save --save-prefix=~
// In the future, this cli will have its own distribution, and isolate from main library.

// Alternatively, run commandline:
// npx pbjs -t static-module -w commonjs -o src/generated/pipeline_spec/pbjs_ml_pipelines.js ../api/v2alpha1/pipeline_spec.proto
// npx pbts -o src/generated/pipeline_spec/pbjs_ml_pipelines.d.ts src/generated/pipeline_spec/pbjs_ml_pipelines.js 
// TODO: Change this file to a shell script for simpler workflow.
const { spawn } = require('child_process');

(async () => {
  const sourceProto = 'api/v2alpha1/pipeline_spec.proto';
  const jsFile = 'frontend/src/generated/pipeline_spec/pbjs_ml_pipelines.js';
  const tsDefFile = 'frontend/src/generated/pipeline_spec/pbjs_ml_pipelines.d.ts';

  // Generate static javascript file for pipeline_spec.
  const pbjsProcess = spawn(
    'pbjs',
    [`-t static-module`, `-w commonjs`, `-o ${jsFile}`, `${sourceProto}`],
    {
      // Allow wildcards in glob to be interpreted
      shell: true,
    },
  );
  pbjsProcess.stdout.on('data', buffer => console.log(buffer.toString()));
  pbjsProcess.stderr.on('data', buffer => console.error(buffer.toString()));
  const code = await new Promise((resolve, reject) => {
    pbjsProcess.on('error', ex => reject(ex));
    pbjsProcess.on('close', code => resolve(code));
  });
  if (code) {
    console.log(`Failed to generate ${jsFile}, error code ${code}`);
    return;
  }
  console.log(`${jsFile} succesfully generated.`);

  // Generate typescript definition file based on javascript file.
  const pbtsProcess = spawn('pbts', [`-o ${tsDefFile}`, `${jsFile}`], {
    // Allow wildcards in glob to be interpreted
    shell: true,
  });
  pbtsProcess.stdout.on('data', buffer => console.log(buffer.toString()));
  pbtsProcess.stderr.on('data', buffer => console.error(buffer.toString()));
  pbtsProcess.on('close', code => {
    if (code) {
      console.log(`Failed to generate ${tsDefFile}, error code ${code}`);
      return;
    }
    console.log(`${tsDefFile} succesfully generated.`);
  });
})();
