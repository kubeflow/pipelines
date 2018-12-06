#!/bin/sh

examples=$(git rev-parse --show-toplevel)/examples

# go to project root
cd "${examples}/generated"
rm -rf cmd models restapi
swagger generate server -A Petstore

cd "${examples}/todo-list"
rm -rf client cmd models restapi
swagger generate client -A TodoList -f ./swagger.yml
swagger generate server -A TodoList -f ./swagger.yml --flag-strategy pflag

cd "${examples}/authentication"
rm -rf client cmd models restapi
swagger generate client -A AuthSample -f ./swagger.yml -P 'models.Principal'
swagger generate server -A AuthSample -f ./swagger.yml -P 'models.Principal'

cd "${examples}/task-tracker"
rm -rf client cmd models restapi
swagger generate client -A TaskTracker -f ./swagger.yml
swagger generate server -A TaskTracker -f ./swagger.yml

cd "${examples}/stream-server"
cp restapi/configure_countdown.go .
rm -rf cmd models restapi
swagger generate server -A Countdown -f ./swagger.yml
mv configure_countdown.go restapi/

cd "${examples}/oauth2"
cp restapi/configure_oauth_sample.go restapi/implementation.go .
rm -rf cmd models restapi
swagger generate server -A oauthSample -P models.Principal -f ./swagger.yml
mv configure_oauth_sample.go implementation.go restapi/

cd "${examples}/tutorials/todo-list/server-1"
rm -rf cmd models restapi
swagger generate server -A TodoList -f ./swagger.yml

cd "${examples}/tutorials/todo-list/server-2"
rm -rf cmd models restapi
swagger generate server -A TodoList -f ./swagger.yml

cd "${examples}/tutorials/todo-list/server-complete"
swagger generate server -A TodoList -f ./swagger.yml

cd "${examples}/tutorials/custom-server"
rm -rf gen
mkdir gen
swagger generate server --exclude-main -A greeter -t gen -f ./swagger/swagger.yml

cd "${examples}/composed-auth"
cp restapi/configure_multi_auth_example.go .
rm -rf cmd models restapi
swagger generate server -A multi-auth-example -P models.Principal -f ./swagger.yml
mv configure_multi_auth_example.go restapi/

cd "${examples}/contributed-templates/stratoscale"
rm -rf client cmd models restapi
swagger generate client -A Petstore --template stratoscale
swagger generate server -A Petstore --template stratoscale

cd ${examples}
go test -vet off -v ./...
