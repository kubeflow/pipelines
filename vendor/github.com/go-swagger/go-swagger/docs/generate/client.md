# Generate an API client from a swagger spec

The toolkit has a command that will let you generate a client.

### Client usage

```
Usage:
  swagger [OPTIONS] generate client [client-OPTIONS]

generate all the files for a client library

Application Options:
  -q, --quiet                                                                     silence logs
  -o, --output=LOG-FILE                                                           redirect logs to file

Help Options:
  -h, --help                                                                      Show this help message

[client command options]
      -f, --spec=                                                                 the spec file to use (default swagger.{json,yml,yaml})
      -a, --api-package=                                                          the package to save the operations (default: operations)
      -m, --model-package=                                                        the package to save the models (default: models)
      -s, --server-package=                                                       the package to save the server specific code (default: restapi)
      -c, --client-package=                                                       the package to save the client specific code (default: client)
      -t, --target=                                                               the base directory for generating the files (default: ./)
      -T, --template-dir=                                                         alternative template override directory
      -C, --config-file=                                                          configuration file to use for overriding template options
      -r, --copyright-file=                                                       copyright file used to add copyright header
          --existing-models=                                                      use pre-generated models e.g. github.com/foobar/model
          --additional-initialism=                                                consecutive capitals that should be considered intialisms
          --with-expand                                                           expands all $ref's in spec prior to generation (shorthand to --with-flatten=expand)
          --with-flatten=[minimal|full|expand|verbose|noverbose|remove-unused]    flattens all $ref's in spec prior to generation (default: minimal, verbose)
      -A, --name=                                                                 the name of the application, defaults to a mangled value of info.title
      -O, --operation=                                                            specify an operation to include, repeat for multiple
          --tags=                                                                 the tags to include, if not specified defaults to all
      -P, --principal=                                                            the model to use for the security principal
      -M, --model=                                                                specify a model to include, repeat for multiple
          --default-scheme=                                                       the default scheme for this client (default: http)
          --default-produces=                                                     the default mime type that API operations produce (default: application/json)
          --skip-models                                                           no models will be generated when this flag is specified
          --skip-operations                                                       no operations will be generated when this flag is specified
          --dump-data                                                             when present dumps the json for the template generator instead of generating files
          --skip-validation                                                       skips validation of spec prior to generation
```

### Build a client

There is an example client provided at: https://github.com/go-swagger/go-swagger/tree/master/examples/todo-list/client

To generate a client:

```
swagger generate client -f [http-url|filepath] -A [application-name] [--principal [principal-name]]
```

If you want to debug what the client is sending and receiving you can set the environment value DEBUG to a non-empty
value.


Use a default client, which has an HTTP transport:

```go
import (
  "log"

  "github.com/myproject/client/operations"
  "github.com/go-openapi/strfmt"
  "github.com/go-openapi/spec"

  apiclient "github.com/myproject/client"
  httptransport "github.com/go-openapi/runtime/client"
)

func main() {

  // make the request to get all items
  resp, err := apiclient.Default.Operations.All(operations.AllParams{})
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("%#v\n", resp.Payload)
}
```

The client runtime allows for a number of [configuration
options](https://godoc.org/github.com/go-openapi/runtime/client#Runtime) to be set.
To then use the client, and override the host, with a HTTP transport:

```go
import (
  "os"
  "log"

  "github.com/myproject/client/operations"
  "github.com/go-openapi/strfmt"
  "github.com/go-openapi/spec"

  apiclient "github.com/myproject/client"
  httptransport "github.com/go-openapi/runtime/client"
)

func main() {

  // create the transport
  transport := httptransport.New(os.Getenv("TODOLIST_HOST"), "", nil)

  // create the API client, with the transport
  client := apiclient.New(transport, strfmt.Default)

  // to override the host for the default client
  // apiclient.Default.SetTransport(transport)

  // make the request to get all items
  resp, err := client.Operations.All(operations.AllParams{})
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("%#v\n", resp.Payload)
}
```

### Authentication

The client supports 3 authentication schemes:

* [Basic Auth](https://godoc.org/github.com/go-openapi/runtime/client#BasicAuth)
* [API key auth in header or query](https://godoc.org/github.com/go-openapi/runtime/client#APIKeyAuth)
* [Bearer token header for oauth2](https://godoc.org/github.com/go-openapi/runtime/client#BearerToken)

```go
import (
  "os"
  "log"

  "github.com/myproject/client/operations"
  "github.com/go-openapi/strfmt"
  "github.com/go-openapi/spec"

  apiclient "github.com/myproject/client"
  httptransport "github.com/go-openapi/runtime/client"
)

func main() {

  // create the API client
  client := apiclient.New(httptransport.New("", "", nil), strfmt.Default)

  // make the authenticated request to get all items
  bearerTokenAuth := httptransport.BearerToken(os.Getenv("API_ACCESS_TOKEN"))
  // basicAuth := httptransport.BasicAuth(os.Getenv("API_USER"), os.Getenv("API_PASSWORD"))
  // apiKeyQueryAuth := httptransport.APIKeyAuth("apiKey", "query", os.Getenv("API_KEY"))
  // apiKeyHeaderAuth := httptransport.APIKeyAuth("X-API-TOKEN", "header", os.Getenv("API_KEY"))
  resp, err := client.Operations.All(operations.AllParams{}, bearerTokenAuth)
  // resp, err := client.Operations.All(operations.AllParams{}, basicAuth)
  // resp, err := client.Operations.All(operations.AllParams{}, apiKeyQueryAuth)
  // resp, err := client.Operations.All(operations.AllParams{}, apiKeyHeaderAuth)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("%#v\n", resp.Payload)
}
```
