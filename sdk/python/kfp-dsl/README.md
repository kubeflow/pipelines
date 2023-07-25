## kfp-dsl package

`kfp-dsl` is a subpackage of the KFP SDK that is released separately in order to provide a minimal dependency runtime package for Lightweight Python Components. **`kfp-dsl` should not be installed and used directly.**

`kfp-dsl` enables the KFP runtime code and objects to be installed at Lightweight Python Component runtime without needing to install the full KFP SDK package.

### Release
`kfp-dsl` should be released immediately prior to each full `kfp` release. The version of `kfp-dsl` should match the version of `kfp` that depends on it.

### Development
To develop on `kfp` with a version of `kfp-dsl` built from source, run the following from the repository root:

```sh
source sdk/python/install_from_source.sh
```

**Note:** Modules in the `kfp-dsl` package are only permitted to have *top-level* imports from the Python standard library, the `typing-extensions` package, and the `kfp-dsl` package itself. Imports from other subpackages of the main `kfp` package or its transitive dependencies must be nested within functions to avoid runtime import errors when only `kfp-dsl` is installed.

### Testing
The `kfp-dsl` code is tested alongside the full KFP SDK in `sdk/python/kfp/dsl-test`. This is because many of the DSL tests require the full KFP SDK to be installed (e.g., requires creating and compiling a component/pipeline).

There are also dedicated `kfp-dsl` tests `./sdk/python/kfp-dsl/runtime_tests/` which test the dedicated runtime code in `kfp-dsl` and should *not* be run with the full KFP SDK installed. Specifically, these tests ensure:
* That KFP runtime logic is correct
* That `kfp-dsl` specifies all of its dependencies (i.e., no module not found errors from missing `kfp-dsl` dependencies)
* That `kfp-dsl` dependencies on the main `kfp` package have associated imports nested inside function calls (i.e., no module not found errors from missing `kfp` dependencies)
