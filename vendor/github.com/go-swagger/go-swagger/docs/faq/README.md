# FAQ

This FAQ is actually a recap of questions reported by the community
(you may search [past and current issues labelled as "question"](https://github.com/go-swagger/go-swagger/issues?q=is%3Aissue+label%3Aquestion)).

Original issues are kept as links for additional details about the inquirer's use-case.

>*We regularly update this document based on questions asked by the community in the "issues" section of the go-swagger repository.*

You may also find most recent questions on Github [here](https://github.com/go-swagger/go-swagger/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3Aquestion).
Feel free to contribute new questions and share your experience with go-swagger!

**Disclaimer**: some of this material might be outdated, as the project is rapidly evolving.

## All contributed questions

<!-- TOC starts here -->

#### Installation,setup and environment
* [What is the minimal go version required?](faq_setup.md#what-is-the-minimal-go-version-required)
<!-- * [Swagger installation issues](faq_setup.md#swagger-installation-issues) -->
<!-- * [What is the proper way to vendor go-swagger?](faq_setup.md#what-is-the-proper-way-to-vendor-go-swagger) -->

#### Model generation
* [Custom validation](faq_model.md#custom-validation)
* [Non-required or nullable property?](faq_model.md#non-required-or-nullable-property)
* [String parameter in body and query?](faq_model.md#string-parameter-in-body-and-query)
* [Request response can have different objects returned based on query parameters](faq_model.md#request-response-can-have-different-objects-returned-based-on-query-parameters)
* [How to validate dates and times?](faq_model.md#how-to-validate-dates-and-times)
* [Accessing the return value from Default response](faq_model.md#accessing-the-default-return-value)
* [How to avoid deep copies of complex data structures that need to be marshalled across the API?](faq_model.md#how-to-avoid-deep-copies-of-complex-data-structures-that-need-to-be-marshalled-across-the-api)
<!-- * How to make custom validators? -->

#### Server generation and customization
* [What are the dependencies required by the generated server?](faq_server.md#what-are-the-dependencies-required-by-the-generated-server)
* [How to add custom flags?](faq_server.md#how-to-add-custom-flags)
* [How do you integrate the flag sets of go-swagger and other packages, in particular, glog?](faq_server.md#how-do-you-integrate-the-flag-sets-of-go-swagger-and-other-packages-in-particular-glog)
* [How to serve two or more swagger specs from one server?](faq_server.md#how-to-serve-two-or-more-swagger-specs-from-one-server)
* [How to access access API struct inside operator handler?](faq_server.md#how-to-access-access-api-struct-inside-operator-handler)
* [Use go-swagger to generate different client or servers](faq_server.md#use-go-swagger-to-generate-different-client-or-servers)
* [Support streaming responses](faq_server.md#support-streaming-responses)

#### Client generation
* [Is there an example for dynamic client?](faq_client.md#example-for-dynamic-client)
* [Can we set a User-Agent header?](faq_client.md#can-we-set-a-user-agent-header)

#### Spec generation from source
* [Is there an example to generate a swagger spec document from the code?](faq_spec.md#example-to-generate-a-swagger-spec-document-from-the-code?)
* [Extra function in example](faq_spec.md#extra-function-in-example)
* [Maps as swagger:parameters](faq_spec.md#maps-as-swagger-parameters)
* [How to define a swagger:response that produces a binary file?](faq_spec.md#how-to-define-a-swagger-response-that-produces-a-binary-file)
* [How to use swagger:params?](faq_spec.md#how-to-use-swagger-params)
* [Empty Definitions as a result?](faq_spec.md#empty-definitions)
* [Documentation / Tutorials?](faq_spec.md#documentation-or-tutorials-on-code-annotation)
* [Wrong schema in response structure?](faq_spec.md#wrong-schema-in-response-structure)
* [go-swagger not generating model info and showing error on swagger UI](faq_spec.md#go-swagger-not-generating-model-info-and-showing-error-on-swagger-ui)
<!--* [Running on google app engine](faq_spec.md#running-on-google-app-engine)-->
<!--* [Generating spec cannot import dependencies](faq_spec.md#generating-spec-cannot-import-dependencies)-->

#### API testing
* [Any suggestions how to leverage the generated client for benchmarking the API?](faq_testing.md#any-suggestions-how-to-leverage-the-generated-client-for-benchmarking-the-api)
* [Using httptest](faq_testing.md#using-httptest)

#### Documenting your API
* [Serving swagger-ui with the API Server](faq_documenting.md#serving-swagger-ui-with-the-api-server)
* [Serving UI from existing app](faq_documenting.md#how-to-serve-swagger-ui-from-a-preexisting-web-app)
* [How to use swagger-ui/cors?](faq_server.md#how-to-use-swagger-ui-cors)
* [Serving my own UI files](faq_server.md#how-to-serve-my-ui-files)

<!-- More on that...
#### Documentation and tutorials
-->
