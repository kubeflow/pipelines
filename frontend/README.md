# Kubeflow Pipelines Management Frontend

## Develop

You need `npm`, install dependencies using `npm install`.

If you made any changes to protos (see backend/README), you'll need to
regenerate the Typescript client library from swagger. We use
swagger-codegen-cli@2.4.7, which you can get
[here](http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.7/swagger-codegen-cli-2.4.7.jar).
Make sure the jar file is somewhere on your path with the name
swagger-codegen-cli.jar, then run `npm run apis`.

You can then do `npm start` to run a static file server at port 3000 that
watches the source files. This also adds a mock backend api server handler to
webpack-dev-server so it can serve basic api calls, as well as a mock
webserver to handle the Single Page App requests, which redirects api
requests to the aforementioned mock api server. For example, requesting the
pipelines page sends a fetch request to
http://localhost:3000/apis/v1beta1/pipelines, which is proxied by the
webserver to the api server at http://localhost:3001/apis/v1beta1/pipelines,
which will return the list of pipelines currently defined in the mock
database.

### Using a real cluster as backend

#### Common steps

1. First configure your `kubectl` to talk to your KFP standalone cluster.
2. `npm start` to start a webpack dev server, it is configured to proxy api requests to localhost:3001. The following step will start a proxy that handles api requests proxied to localhost:3001.

#### Special step that depend on what you want to do

| What to develop?        | Who handles API requests? | Script to run                                                  | Extra notes                                                        |
| ----------------------- | ------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------ |
| Client UI               | standalone deployment     | `NAMESPACE=kubeflow npm run start:proxy-standalone`            |                                                                    |
| Client UI + Node server | standalone deployment     | `NAMESPACE=kubeflow npm run start:proxy-standalone-and-server` | You need to rerun the script every time you edit node server code. |

TODO: figure out and document how to use a Kubeflow deployment to develop UI.

**Production Build:**
You can do `npm run build` to build the frontend code for production, which
creates a ./build directory with the minified bundle. You can test this bundle
using `server/server.js`. Note you need to have an API server running, which
you can then feed its address (host + port) as environment variables into
`server.js`. See the usage instructions in that file for more.

The mock API server and the mock webserver can still be used with the
production UI code by running `npm run mock:api` and `npm run mock:server`.

**Container Build:**

You can also do `npm run docker` if you have docker installed to build an
image containing the production bundle and the server pieces. In order to run
this image, you'll need to port forward 3000, and pass the environment
variables `ML_PIPELINE_SERVICE_HOST` and
`ML_PIPELINE_SERVICE_PORT` with the details of the API server, which
you can run using `npm run api` separately.

## Code Style

We use [prettier](https://prettier.io/) for code formatting, our prettier config
is [here](https://github.com/kubeflow/pipelines/blob/master/frontend/.prettierrc.yaml).

To understand more what prettier is: [What is Prettier](https://prettier.io/docs/en/index.html).

### IDE Integration

- For vscode, install the plugin "Prettier - Code formatter" and it will pick
  this project's config automatically.
  Recommend setting the following in [settings.json](https://code.visualstudio.com/docs/getstarted/settings#_settings-file-locations) for vscode to autoformat + organize import on save.
  ```
  "[typescript]": {
      "editor.codeActionsOnSave": {
          "source.organizeImports": true,
      },
      "editor.formatOnSave": true,
  },
  "[typescriptreact]": {
      "editor.codeActionsOnSave": {
          "source.organizeImports": true,
      },
      "editor.formatOnSave": true,
  },
  ```
- For others, refer to https://prettier.io/docs/en/editors.html

### Format Code Manually

Run `npm run format`.

### Escape hatch

If there's some code that you don't want being formatted by prettier, follow
guide [here](https://prettier.io/docs/en/ignore.html). (Most likely you don't need this.)
