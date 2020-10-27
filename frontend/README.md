# Kubeflow Pipelines Management Frontend

## Tools you need

You need `node v12` and `npm`.
Recommend installing `node` and `npm` using https://github.com/nvm-sh/nvm. After installing nvm,
you can install `node v12` by `nvm install 12`.

## Manage dev environment with npm

### First time
1. Clone this repo
2. Navigate to frontend folder: `cd $KFP_SRC/frontend`.
3. Install dependencies: `npm ci`.

`npm ci` makes sure your installed dependencies have the exact same version as others. (Usually, you just
need to run this once, but after others installed extra dependencies, you need to run `npm ci` again to
get package updates.)

### Package management
Run `npm install --save <package>` (or `npm i -S <package>` for short) to install runtime dependencies and save them to package.json.
Run `npm install --save-dev <package>` (or `npm i -D <package>` for short) to install dev dependencies and save them to package.json.

To upgrade @kubeflow/frontend, run `npm i -S kubeflow/frontend#<commit-hash>`. Get the commit hash from https://github.com/kubeflow/frontend/commits/master.

### Daily workflow
You will see a lot of `npm run xxx` commands in instructions below, the actual script being run is defined in the "scripts" field of [package.json](https://github.com/kubeflow/pipelines/blob/91db95a601fa7fffcb670cb744a5dcaeb08290ae/frontend/package.json#L32). Development common scripts are maintained in package.json, and we use npm to call them conveniently.

### npm next step
You can learn more about npm in https://docs.npmjs.com/about-npm/.

## Start frontend development server

You can then do `npm start` to run a webpack dev server at port 3000 that
watches the source files. It also redirects api requests to localhost:3001. For
example, requesting the pipelines page sends a fetch request to
http://localhost:3000/apis/v1beta1/pipelines, which is proxied by the
webserver to http://localhost:3001/apis/v1beta1/pipelines,
which should return the list of pipelines.

Follow the next section to start an api mock/proxy server to let localhost:3001
respond to api requests.

## Start api mock/proxy server

### Api mock server

This is the easiest way to start developing, but it does not support all apis during
development.

Run `npm run mock:api` to start a mock backend api server handler so it can
serve basic api calls with mock data. 

### Proxy to a real cluster

This requires you already have a real KFP cluster, you can proxy requests to it.

Before you start, configure your `kubectl` to talk to your KFP cluster.

Then it depends on what you want to develop:

| What to develop?        | Script to run                                                  | Extra notes                                                        |
| ----------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------ |
| Client UI               | `NAMESPACE=kubeflow npm run start:proxy`            |                                                                    |
| Client UI + Node server | `NAMESPACE=kubeflow npm run start:proxy-and-server` | You need to rerun the script every time you edit node server code. |
| Client UI + Node server (debug mode) | `NAMESPACE=kubeflow npm run start:proxy-and-server-inspect` | Same as above, and you can use chrome to debug the server. |

## Unit testing FAQ
There are a few typees of tests during presubmit:
* formatting, refer to [Code Style Section](#code-style)
* linting, you can also run locally with `npm run lint`
* client UI unit tests, you can run locally with `npm test`
* UI node server unit tests, you can run locally with `cd server && npm test`

There is a special type of unit test called [snapshot tests](https://jestjs.io/docs/en/snapshot-testing). When
snapshot tests are failing, you can update them automatically with `npm test -u` and run all tests. Then commit
the snapshot changes.

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

## Code Style

We use [prettier](https://prettier.io/) for code formatting, our prettier config
is [here](https://github.com/kubeflow/pipelines/blob/master/frontend/.prettierrc.yaml).

To understand more what prettier is: [What is Prettier](https://prettier.io/docs/en/index.html).

### IDE Integration

- For vscode, install the plugin "Prettier - Code formatter" and it will pick
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
- For others, refer to https://prettier.io/docs/en/editors.html.

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
Make sure the jar file is somewhere on your path with the name
swagger-codegen-cli.jar, then run `npm run apis`.
