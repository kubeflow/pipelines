# KFP Release tools

To do a release:

TAG=`<TAG>` BRANCH=`<BRANCH>` make release

Example, release in release branch:

```bash
    TAG=1.4.0 BRANCH=release-1.4 make release
```

Example, release an RC (release candidate) in master branch:

```bash
    TAG=2.0.0-rc.1 BRANCH=master make release
```

## Dev guide

Take a look at the `./Makefile`, all the rules there are entrypoints for dev activities.
