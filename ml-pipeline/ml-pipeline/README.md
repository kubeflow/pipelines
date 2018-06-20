# ml-pipeline

> Prototypes for running ML pipeline.


* [Quickstart](#quickstart)
* [Using Prototypes](#using-prototypes)

## Quickstart

*The following commands use the `io.ksonnet.pkg.ml-pipeline` prototype to generate Kubernetes YAML for ml-pipeline, and then deploys it to your Kubernetes cluster.*

First, create a cluster and install the ksonnet CLI (see root-level [README.md](rootReadme)).

If you haven't yet created a [ksonnet application](linkToSomewhere), do so using `ks init <app-name>`.

Finally, in the ksonnet application directory, run the following:

```shell
# Expand prototype as a Jsonnet file, place in a file in the
# `components/` directory. (YAML and JSON are also available.)
$ ks prototype use io.ksonnet.pkg.ml-pipeline ml-pipeline \
  --name ml-pipeline \
  --namespace default

# Apply to server.
$ ks apply -f ml-pipeline.jsonnet
```

## Using the library

The library files for ml-pipeline define a set of relevant *parts* (_e.g._, deployments, services, secrets, and so on) that can be combined to configure ml-pipeline for a wide variety of scenarios. For example, a database like Redis may need a secret to hold the user password, or it may have no password if it's acting as a cache.

This library provides a set of pre-fabricated "flavors" (or "distributions") of ml-pipeline, each of which is configured for a different use case. These are captured as ksonnet *prototypes*, which allow users to interactively customize these distributions for their specific needs.

These prototypes, as well as how to use them, are enumerated below.



[rootReadme]: https://github.com/ksonnet/mixins