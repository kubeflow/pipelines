# Proto backwards compatibility testing

This folder contains tests that verify that newly generated proto files do not break backwards compatibility with proto binary message translation and JSON unmarshalling. 

The structure is as follows: 

## objects.go 
These contain the Go Structs that use the latest proto generated Go Code. 

## testdata/generated-commit 
These files contain the proto binary messages and the JSON unmarshalled data using proto generated files found in the commit: `<commit>`. 

This allows us to have a ground truth to compare the newly generated code against. The tests will unmarshal the proto binaries from `<commit>` and compare it with the structs created using new generated code. 

In much the same way we also test for JSON unmarshalling. You will notice that we adjust the unmarshalling options for JSON to match what we use in the grpc-gateway unmarshalling configuration for the grpc server. 

## Updating generate code

Generated code can be updated by running the test code with `UPDATE_EXPECTED=true` environment variable set in your commandline. For example: 

```shell
cd backend/test/proto_tests
export UPDATE_EXPECTED=true
go test . # update generate code
export UPDATE_EXPECTED=false
go test . # verify your changes
```

Note that it is very unlikely you should need to update this code, if you do then it is a good sign you are introducing breaking changes, so use it wisely. 
