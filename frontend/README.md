# ML Pipeline Management Frontend

**Develop:** You need `npm`, install dependencies using `npm install`.

You can then do `npm run dev` to run a static file server at port 3000 that
watches the source files, rebuilds into `./dist` and serves the latest changes
from there. This also adds a mock backend server handler to webpack-dev-server
so it can serve basic api calls. For example: http://localhost:3000/_api/packages
will return the list of packages currently defined in the mock database.
