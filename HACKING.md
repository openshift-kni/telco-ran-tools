# Developing in telco-ran-tools

## Makefile targets

## Linter tests

As of this writing, two linters are run by the `make ci-job` command:

* shellcheck
* bashate

These tests will be run automatically as part of the ci-job test post a pull request an update. Failures will mean your pull request
cannot be merged. It is recommended that you run `make ci-job` regularly as part of development.

## Updates to extract-ai.sh and extract-ocp.sh

When updating the extract-ai.sh and extract-ocp.sh scripts, you will also need to update the ignition sample files and
the corresponding documentation. This can be done by running:
```bash
make update-resources
```

## Building the image

The Makefile has variables to support building images for development:
* REPOOWNER - the registry in quay.io for the image (defaults to openshift-kni)
* IMAGENAME - the image name (defaults to telco-ran-tools)
* IMAGETAG - the image tag (defaults to latest)

To build the image, run the `image` make target, or run the `push` make target to build and push the image:
```bash
make REPOOWNER=${USER} push
```
