# Authentication sample

Generate the code with a security principal:

```shell
swagger generate server -A AuthSample -P models.Principal -f ./swagger.yml
```

Edit the ./restapi/configure_auth_sample.go file

```go
func configureAPI(api *operations.AuthSampleAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "x-token" header is set
	api.KeyAuth = func(token string) (*models.Principal, error) {
		if token == "abcdefuvwxyz" {
			prin := models.Principal(token)
			return &prin, nil
		}
		api.Logger("Access attempt with incorrect api key auth: %s", token)
		return nil, errors.New(401, "incorrect api key auth")
	}

	api.CustomersCreateHandler = customers.CreateHandlerFunc(func(params customers.CreateParams, principal *models.Principal) middleware.Responder {
		return middleware.NotImplemented("operation customers.Create has not yet been implemented")
	})
	api.CustomersGetIDHandler = customers.GetIDHandlerFunc(func(params customers.GetIDParams, principal *models.Principal) middleware.Responder {
		return middleware.NotImplemented("operation customers.GetID has not yet been implemented")
	})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}
```

Run the server:

```shell
go run ./cmd/auth-sample-server/main.go --port 35307
```

Exercise auth:

```shellsession
± ivan@avalon:~  
 » curl -i -H 'Content-Type: application/keyauth.api.v1+json' -H 'X-Token: abcdefuvwxyz' http://127.0.0.1:35307/api/customers
```
```http
HTTP/1.1 501 Not Implemented
Content-Type: application/keyauth.api.v1+json
Date: Fri, 25 Nov 2016 19:14:14 GMT
Content-Length: 57

"operation customers.GetID has not yet been implemented"
```
```shellsession
± ivan@avalon:~  
 » curl -i -H 'Content-Type: application/keyauth.api.v1+json' -H 'X-Token: abcdefu' http://127.0.0.1:35307/api/customers
```
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/keyauth.api.v1+json
Date: Fri, 25 Nov 2016 19:16:49 GMT
Content-Length: 47

{"code":401,"message":"incorrect api key auth"}       
```
