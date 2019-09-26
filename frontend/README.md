# Kubeflow Pipelines Management Frontend

**Develop:**
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

1. First configure your `kubectl` to talk to your kfp lite cluster.
2. `npm run start:proxies` to start proxy servers that port forwards to your cluster.
3. `npm start` to start a webpack dev server, it has already been configured to talk to aforementioned proxies.

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
