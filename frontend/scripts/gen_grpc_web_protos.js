// NOTE: This script is expected to run from root directory.
// But the frontend/package.json has handled the path navigation for you.
// Make sure you are under path <parent_directory>/pipelines/frontend.
// npm command: npm run build:protos
const fs = require('fs');
const path = require('path');
const {spawn} = require('child_process');

// TODO: Build process should remove the existing generated proto definitions.
const OUT_DIR = path.join(__dirname, '..', 'src', 'mlmd', 'generated');

if (!fs.existsSync(OUT_DIR)) {
  fs.mkdirSync(OUT_DIR, {
    recursive: true
  });
}
console.log(`Generating PROTOS in: ${OUT_DIR}`);

// Expects protoc to be on your PATH.
// From npm/google-protobuf:
// The compiler is not currently available via npm, but you can download a
// pre-built binary on GitHub (look for the protoc-*.zip files under Downloads).
// proto_path and target is relative to <parent_directory>/pipelines.
const protocProcess = spawn(
    'protoc', [
      `--js_out="import_style=commonjs,binary:${OUT_DIR}"`,
      `--grpc-web_out="import_style=commonjs+dts,mode=grpcweb:${OUT_DIR}"`,
      `--proto_path="./third_party/ml-metadata"`,
      'third_party/ml-metadata/ml_metadata/**/*.proto'
    ], {
      // Allow wildcards in glob to be interpreted
      shell: true
    }
);
protocProcess.stdout.on('data', buffer => console.log(buffer.toString()));
protocProcess.stderr.on('data', buffer => console.error(buffer.toString()));
protocProcess.on('close', code => {
  if (code) return;
  console.log(`Protos succesfully generated.`)
});
