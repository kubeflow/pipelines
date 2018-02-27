# ML Pipeline Management Frontend

**Develop:** You need `npm`, install dependencies using `npm install`.

You can then do `npm run dev` to run a static file server at port 3000 that
watches the source files, rebuilds into `./dist` and serves the latest changes
from there. This also adds a mock backend server handler to webpack-dev-server
so it can serve basic api calls. For example: http://localhost:3000/_api/packages
will return the list of packages currently defined in the mock database.

**Production Build:**
You can do `npm run build` to build the frontend code for production, which creates a ./dist directory with the minified bundle. You can test this bundle using `server/server.js`.

You can also do `npm run docker` if you have docker installed to build an image containing the production bundle and the server pieces. In order to run this image, you'll need to port forward 3000, and pass the environment variable `API_SERVER_ADDRESS` with the address of the API server, which you can run using `npm run api` separately.
