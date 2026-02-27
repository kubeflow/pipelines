# Kubeflow Pipelines Frontend Contributing Guide

## Node setup

Install the node version specified in the [.nvmrc](.nvmrc) file. You
can use [nvm](https://github.com/nvm-sh/nvm) or
[fnm](https://github.com/Schniz/fnm) to manage your node installations; fnm is
basically a faster version of nvm implemented in Rust. Installation instructions
are available at their corresponding GitHub repositories.

### fnm

```bash
fnm install 22.19.0
fnm use 22.19.0
```

### nvm

```bash
nvm install 22.19.0
nvm use 22.19.0
```

## Manage dev environment with npm

### Pre-requisites

Clone the Kubeflow Pipelines repo:

```bash
# Set this to your working directory
WORKING_DIRECTORY= ...
git clone https://github.com/kubeflow/pipelines.git ${WORKING_DIRECTORY}
```

Navigate to frontend folder:

```bash
cd ${WORKING_DIRECTORY}/frontend
```

Install the NPM dependencies:

`npm ci`.

`npm ci` ensures exact dependency versions are installed according to `package-lock.json`. You should run it:
- When first setting up the project
- After `package.json` or `package-lock.json` changes
- In CI/CD pipelines

> [!Note]
> This command deletes the existing `node_modules` folder and requires a `package-lock.json` file.

### Daily workflow

You will see a lot of `npm run xxx` commands in the instructions below, the actual script being run is defined in the "scripts" field of [package.json](https://github.com/kubeflow/pipelines/blob/91db95a601fa7fffcb670cb744a5dcaeb08290ae/frontend/package.json#L32). Common development scripts are maintained in package.json, and we use npm to call them conveniently.

### npm next step

You can learn more about npm in https://docs.npmjs.com/about-npm/.

## Start frontend development server

You can then do `npm start` to run a Vite dev server at port 3000 that
watches the source files. It also redirects api requests to localhost:3001. For
example, requesting the pipelines page sends a fetch request to
http://localhost:3000/apis/v1beta1/pipelines, which is proxied by the
webserver to http://localhost:3001/apis/v1beta1/pipelines,
which should return the list of pipelines. To override the port, run
`npm run start -- --port 3002` or update `frontend/vite.config.mts`.

Follow the next section to start an API server (mock or proxy) to let localhost:3001
respond to API requests.

## Start api mock/proxy server

### Api mock server

This is the easiest way to start developing, but it does not support all apis during
development.

Run `npm run mock:api` to start a mock backend api server handler so it can
serve basic api calls with mock data.

If you want to port real MLMD store to be used for mock backend scenario, you can run the following command. Note that a mock MLMD store doesn't exist yet.

```
kubectl port-forward svc/metadata-envoy-service 9090:9090
```

### Proxy to a real cluster

You can proxy requests from the UI running on your host machine to an actual KFP
deployment running on a remote or local Kubernetes cluster. This dramatically
improves iteration time, especially since the docker build can take 20+ minutes.

KFP can be deployed in single-user or multi-user mode. Since there's a delta in
logic between between the two modes, automated tests and manual validation
against a single-user cluster can still fail when deployed to a multi-user
cluster.

#### Single-user

1. Deploy a standalone KFP instance by running the following:
```bash
git clone https://github.com/kubeflow/pipelines.git ${WORKING_DIRECTORY}
cd ${WORKING_DIRECTORY}/backend
make kind-cluster-agnostic
```
2. Use the following table to determine which script to run.

| What to develop?                     | Script to run                            | Extra notes                                                        |
|--------------------------------------|------------------------------------------|--------------------------------------------------------------------|
| Client UI                            | `npm run start:proxy`                    |                                                                    |
| Client UI + Node server              | `npm run start:proxy-and-server`         | You need to rerun the script every time you edit node server code. |
| Client UI + Node server (debug mode) | `npm run start:proxy-and-server-inspect` | Same as above, and you can use chrome to debug the server.         |

#### Multi-user

1. Install Kubernetes and deploy KFP to it on your your local machine by
   following the [multi-user Kubeflow installation instructions](https://github.com/kubeflow/manifests?tab=readme-ov-file#installation).
2. Run `cd frontend`.
3. Run the following code block.

    ```bash
    export VITE_NAMESPACE=kubeflow-user-example-com
    npm run build
    ```

   If you're targeting the cluster installed in step 1, the target namespace
   defaults to `kubeflow-user-example-com`. If you're targeting a different
   cluster / namespace, make sure to update the `VITE_NAMESPACE`
   environment variable.
4. Install
   [mod-header](https://chromewebstore.google.com/detail/modheader-modify-http-hea/idgpnmonknjnojddfkpgkljpfnnfcklj?hl=en)
   for Chrome.
5. Open the mod-header modal.
6. If you're targeting the cluster installed in step 1, add a header with a key
   of `kubeflow-userid` and a value of `user@example.com`. If you're targeting a
   different cluster / namespace, add the corresponding userid / namespace.
7. Add `localhost:3001` to the filter field in the modal.
8. Run the commands from the [Single-user](#single-user) section above, e.g.
   `npm run start:proxy-and-server`.
9. Navigate to http://localhost:3001. The UI running on your host machine should
   now be able to communicate to a KFP backend deployed in multi-user mode.
10. If you want the local UI server to target a single-user cluster again,
    you'll need to run the following first:

    ```bash
    unset VITE_NAMESPACE
    npm run build
    ```

## Unit testing FAQ

There are a few types of tests during pre-submit:

* formatting, refer to [Code Style Section](#code-style)
* linting, you can also run locally with `npm run lint`
* TypeScript typecheck (no emit), run locally with `npm run typecheck`
* client UI unit tests (Vitest), you can run locally with `npm run test:ui`
  (uncapped workers) or `npm run test:ui:coverage:loop` for stability loops
  (coverage + `--maxWorkers 4`). `npm test` is an alias for `vitest run`.
* UI node server unit tests (Jest), you can run locally with
  `npm run test:server:coverage` or `cd server && npm test -- --coverage`

There is a special type of unit test called [snapshot tests](https://jestjs.io/docs/en/snapshot-testing). When
snapshot tests are failing, you can update them automatically with `npm test -u` or
`npm run test:ui -- -u` (Vitest) and run all tests. For server test snapshots (if any),
use `cd server && npm test -- -u`. Then commit the snapshot changes.

## Production Build

You can do `npm run build` to build the frontend code for production, which
creates a ./build directory with the minified bundle. You can test this bundle
using `server/server.js`. Note you need to have an API server running, which
you can then feed its address (host + port) as environment variables into
`server.js`. See the usage instructions in that file for more.

## Container Build

You can also do `npm run docker` if you have docker installed to build an
image containing the production bundle and the server pieces. In order to run
this image, you'll need to port forward 3000, and pass the environment
variables `ML_PIPELINE_SERVICE_HOST` and
`ML_PIPELINE_SERVICE_PORT` with the details of the API server.

### Package management

Run `npm install --save <package>` (or `npm i -S <package>` for short) to install runtime dependencies and save them to package.json.
Run `npm install --save-dev <package>` (or `npm i -D <package>` for short) to install dev dependencies and save them to package.json.

## Code Style

We use [prettier](https://prettier.io/) for code formatting, our prettier config
is [here](https://github.com/kubeflow/pipelines/blob/master/frontend/.prettierrc.yaml).

To understand more what prettier is: [What is Prettier](https://prettier.io/docs/en/index.html).

### IDE Integration

* For vscode, install the plugin "Prettier - Code formatter" and it will pick
  this project's config automatically.
  Recommend setting the following in [settings.json](https://code.visualstudio.com/docs/getstarted/settings#_settings-file-locations) for vscode to autoformat on save.
  ```
  "[typescript]": {
    "editor.formatOnSave": true,
    "files.trimTrailingWhitespace": false,
  },
  "[typescriptreact]": {
    "editor.formatOnSave": true,
    "files.trimTrailingWhitespace": false,
  },
  ```
  Also, vscode builtin trailing whitespace [conflicts with jest inline snapshot](https://github.com/Microsoft/vscode/issues/52711), so recommend disabling it.
* For others, refer to https://prettier.io/docs/en/editors.html.

### Format Code Manually

Run `npm run format`.

### Escape hatch

If there's some code that you don't want being formatted by prettier, follow
guide [here](https://prettier.io/docs/en/ignore.html). (Most likely you don't need this.)

## Api client code generation

If you made any changes to protos (see backend/README), you'll need to
regenerate the Typescript client library from swagger. We use
swagger-codegen-cli@2.4.7, which you can get
[here](https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.7/).
Make sure to add the jar file to $PATH with the name swagger-codegen-cli.jar, then run `npm run apis` for
v1 api or `npm run apis:v2beta1` for v2 api.
```
// add jar file to $PATH
JAR_PATH=<folder-path-to-jar-file>
export PATH="$JAR_PATH:$PATH"
```
After code generation, you should run `npm run format` to format the output and avoid creating a large PR.

## MLMD components

* `src/mlmd` - components for visualizing data from an `ml-metadata` store. For more information see the
  [google/ml-metadata](https://github.com/google/ml-metadata) repository.

This module previously lived in [kubeflow/frontend](https://github.com/kubeflow/frontend) repository. It contains tsx files for visualizing MLMD components.

MLMD protos lives in `pipelines/third_party/ml-metadata/ml_metadata/`, and the generated JS files live in `pipelines/frontend/src/third_party/mlmd`.

### Building generated metadata Protocol Buffers

* `build:protos` - for compiling Protocol Buffer definitions


This project contains a mix of natively defined classes and classes generated by the Protocol
Buffer Compiler from definitions in the [pipelines/third_party/ml-metadata/ml_metadata/](third_party/ml-metadata/ml_metadata/) directory. Copies of the generated classes are
included in the [pipelines/frontend/src/third_party/mlmd](frontend/src/third_party/mlmd) directory to allow the build process to succeed without a dependency on
the Protocol Buffer compiler, `protoc`, being in the system PATH.

If a file in [pipelines/third_party/ml-metadata/ml_metadata/proto](third_party/ml-metadata/ml_metadata/proto) is modified or you need to manually re-generate the protos, you'll need to:

* Add `protoc` ([download](https://github.com/protocolbuffers/protobuf/releases)) to your system
  PATH

  ```bash
  # Example:
  apt install -y protobuf-compiler=3.15.8
  ```

* Add `protoc-gen-grpc-web` ([download](https://github.com/grpc/grpc-web/releases)) to your system
  PATH

  ```bash
  # Example:
  curl -LO https://github.com/grpc/grpc-web/releases/download/1.4.2/protoc-gen-grpc-web-1.4.2-linux-x86_64
  mv protoc-gen-grpc-web-1.4.2-linux-x86_64 /usr/local/bin/protoc-gen-grpc-web
  chmod +x /usr/local/bin/protoc-gen-grpc-web
  ```

* Generate new protos by running

  ```bash
  npm run build:protos
  ```

The script run by `npm run build:protos` can be found at `scripts/gen_grpc_web_protos.js`.

The current TypeScript proto library was generated with `protoc-gen-grpc-web` version 1.2.1 with
`protoc` version 3.17.3.

The Protocol Buffers in [pipelines/third_party/ml-metadata/ml_metadata/proto](third_party/ml-metadata/ml_metadata/proto) are taken from the target version(v1.0.0 by default) of the `ml_metadata` proto
package from
[google/ml-metadata](https://github.com/google/ml-metadata/tree/master/ml_metadata/proto).

## Pipeline Spec (IR) API

For KFP v2, we use pipeline spec or IR(Intermediate Representation) to represent a Pipeline definition. It is saved as json payload when transmitted. You can find the API in [api/v2alpha1/pipeline_spec.proto](api/v2alpha1/pipeline_spec.proto). To take the latest of this file and compile it to Typescript classes, follow the below step:

```
npm run build:pipeline-spec
```

See the explanation on what it does below:

### Convert buffer to a runtime object using protoc

Prerequisite: Add `protoc` ([download](https://github.com/protocolbuffers/protobuf/releases)) to your system PATH

Compile pipeline_spec.proto to Typed classes in TypeScript,
so it can convert a payload stream to a PipelineSpec object during runtime.

You can check out the result like `pipeline_spec_pb.js`, `pipeline_spec_pb.d.ts` in [frontend/src/generated/pipeline_spec](/frontend/src/generated/pipeline_spec).

The plugin tool for conversion we currently use is [ts-proto](https://github.com/stephenh/ts-proto). We previously use
[protobuf.js](https://github.com/protobufjs/protobuf.js) but it doesn't natively support Protobuf.Value processing.

You can checkout the generated TypeScript interfaces in [frontend/src/generated/pipeline_spec/pipeline_spec.ts](/frontend/src/generated/pipeline_spec/pipeline_spec.ts)

<!-- ARCHIVE: We switched to use ts-proto for now.
### Encode plain object to buffer using protobuf.js

protoc doesn't provide a way to convert plain object to
payload stream, therefore we need a helper tool `protobuf.js` to validate and encode plain object.

You can check out the result like `pbjs_ml_pipelines.js`, `pbjs_ml_pipelines.d.ts` in [frontend/src/generated/pipeline_spec](frontend/src/generated/pipeline_spec).
-->

## Platform Spec API
For KFP v2, we use platform spec to represent a platform definition.

### Kubernetes
The details of Kubernetes platform is in [kubernetes_platform/proto/kubernetes_executor_config.proto](kubernetes_platform/proto/kubernetes_executor_config.proto). To take the latest of this file and compile it to Typescript classes, follow the below step:

```
npm run build:platform-spec:kubernetes-platform
```

## Large features development

To accommodate KFP v2 development, we create a `frontend feature flag` capability which hides features under development behind a flag. Only when developer explicitly enables these flags, they can see those features. To control the visibility of these features, check out a webpage similar to pattern http://localhost:3000/#/frontend_features.

To manage feature flags default values, visit [frontend/src/feature.ts](frontend/src/feature.ts) for `const features`. To apply the default feature flags locally in your browser, run `localStorage.setItem('flags', "")` in browser console.

## Storybook

For component driven UI development, KFP UI integrates with Storybook to develop v2 features. To run Storybook locally, run `npm run storybook` and visit `localhost:6006` in browser.
