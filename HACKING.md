# Developing in telco-ran-tools

- [Developing in telco-ran-tools](#developing-in-telco-ran-tools)
  - [Makefile targets](#makefile-targets)
  - [Linter tests](#linter-tests)
  - [Updates to extract-ai.sh and extract-ocp.sh](#updates-to-extract-aish-and-extract-ocpsh)
  - [Building the image](#building-the-image)
  - [Regression testing](#regression-testing)

## Makefile targets

## Linter tests

As of this writing, two linters are run by the `make ci-job` command:

- shellcheck
- bashate

These tests will be run automatically as part of the ci-job test post a pull request an update. Failures will mean your pull request
cannot be merged. It is recommended that you run `make ci-job` regularly as part of development.

Additionally, markdownlint-cli2 can be run manually using `make markdownlint`.

## Updates to extract-ai.sh and extract-ocp.sh

When updating the extract-ai.sh and extract-ocp.sh scripts, you will also need to update the ignition sample files and
the corresponding documentation. This can be done by running:

```bash
make update-resources
```

## Building the image

The Makefile has variables to support building images for development:

- RUNTIME - local container management too (defaults to podman)
- DOCKERFILE - Name of Docker file for build (defaults to Dockerfile)
- REPOOWNER - the registry in quay.io for the image (defaults to openshift-kni)
- IMAGENAME - the image name (defaults to telco-ran-tools)
- IMAGETAG - the image tag (defaults to latest)

To build the image, run the `image` make target, or run the `push` make target to build and push the image:

```bash
make REPOOWNER=${USER} push
```

To build the image using the Dockerfile.dev file, which uses the publicly accessible fedora base image, run:

```bash
make REPOOWNER=${USER} DOCKERFILE=Dockerfile.dev push
```

## Regression testing

A regression test suite utility is built into the image. It requires the pull secrets in `/root/.docker/config.json`, similar to
the `download` command, as it will run `oc-mirror`, but it will avoid downloading the images. New tests should be added in the regression
folder as needed. In addition, developers should run the regression tests prior to posting a pull request.

```console
[root@host core]# time podman run -v /var/lib/kubelet/config.json:/root/.docker/config.json:Z --rm quay.io/openshift-kni/telco-ran-tools:latest -- regression-suite.sh
################################################################################
Mon May 29 13:25:28 UTC 2023: Running test: du-profile
################################################################################
Mon May 29 13:30:02 UTC 2023: Running test: generate-imageset
################################################################################
Mon May 29 13:30:02 UTC 2023: Running test: invalid-acm-version-format
################################################################################
Mon May 29 13:30:02 UTC 2023: Running test: invalid-mce-version-format
################################################################################
Mon May 29 13:30:02 UTC 2023: Running test: invalid-version-format
################################################################################
Mon May 29 13:30:02 UTC 2023: Running test: keep-stale
Stale file found: /mnt/testsuite/not-a-real-image@sha256_1234567890123456789012345678901234567890123456789012345678901234.tgz
################################################################################
Mon May 29 13:32:58 UTC 2023: Running test: stale-cleanup
################################################################################
Mon May 29 13:35:54 UTC 2023: Running test: unavailable-acm-and-mce-versions
################################################################################
Mon May 29 13:37:53 UTC 2023: Running test: unavailable-acm-version
################################################################################
Mon May 29 13:39:49 UTC 2023: Running test: unavailable-mce-version
################################################################################
Mon May 29 13:40:48 UTC 2023: Running test: unknown-option
################################################################################

Test Results:

Test Name                                 Status
========================================  ==========
du-profile                                PASSED
generate-imageset                         PASSED
invalid-acm-version-format                PASSED
invalid-mce-version-format                PASSED
invalid-version-format                    PASSED
keep-stale                                PASSED
stale-cleanup                             PASSED
unavailable-acm-and-mce-versions          PASSED
unavailable-acm-version                   PASSED
unavailable-mce-version                   PASSED
unknown-option                            PASSED

Number of tests run:  11
Total passed:         11
Total failed:          0

real    15m22.912s
user    0m1.519s
sys     0m0.667s
[root@host core]#
```
