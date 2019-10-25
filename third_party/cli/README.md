# CLI tools to fetch license info

## Why we need this?

When we release third party images (can be considered as redistributing third
party binary), we should be compliant to their licenses. Not just the library's
license, also its dependencies and transitive dependencies' licenses.

We need to do the following to be compliant:
* Put license declarations in the image for all licenses.
* Mirror source code in the image for code with MPL, EPL, GPL or CDDL licenses.

It's not an easy task to get license of all (transitive) dependencies of a go
library. Thus, we need these tools to automate this task.

## How to get all dependencies with license and source code?

1. Install CLI tools here: `python setup.py install`
1. Collect dependencies + transitive dependencies in a go library. Put them together in a text file called `dep.txt`. Format: each line has a library name. The library name should be a valid golang import module name.

    Example ways to get it:
    * argo uses gopkg for package management. It has a [Gopkg.lock file](https://github.com/argoproj/argo/blob/master/Gopkg.lock)
    with all of its dependencies and transitive dependencies. All the name fields in this file is what we need. You can run `parse-toml-dep` to parse it.
    * minio uses [official go modules](https://blog.golang.org/using-go-modules), there's a [go.mod file](https://github.com/minio/minio/blob/master/go.mod) describing its direct dependencies. Run command `go list -m all` to get final versions that will be used in a build for all direct and indirect dependencies, [reference](https://github.com/golang/go/wiki/Modules#daily-workflow). Parse its output to make a file we need.

    Reminder: don't forget to put the library itself into `dep.txt`.
1. Run `get-github-repo` to resolve github repos of golang imports. Not all
imports can be figured out by my script, needs manual help for <2% of libraries.

    For a library we cannot resolve, manually put it in `dep-repo-mapping.manual.csv`, so the tool knows how to find its github repo the next time.

    Defaults to read dependencies from `dep.txt` and writes to `repo.txt`.
1. Run `get-github-license-info` to crawl github license info of these libraries. (Not all repos have github recognizable license, needs manual help for <2% of libraries)

    Defaults to read repos from `repo.txt` and writes to `license-info.csv`. You
    need to configure github personal access token because it sends a lot of
    requests to github. Follow instructions in `get-github-license-info -h`.

    For repos that fails to fetch license, it's usually because their github repo
    doesn't have a github understandable license file. Check its readme and
    update correct info into `license-info.csv`. (Usually, use its README file which mentions license.)
1. Edit license info file. Manually check the license file for all repos with a license categorized as "Other" by github. Figure out their true license names.
1. Run `concatenate-license` to crawl full text license files for all dependencies and concat them into one file.

    Defaults to read license info from `license-info.csv`. Writes to `license.txt`.
    Put `license.txt` to `third_party/library/license.txt` where it is read when building docker images.
1. Manually update a list of dependencies that requires source code, put it into `third_party/library/repo-MPL.txt`.
