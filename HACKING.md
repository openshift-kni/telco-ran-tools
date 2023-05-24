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

- REPOOWNER - the registry in quay.io for the image (defaults to openshift-kni)
- IMAGENAME - the image name (defaults to telco-ran-tools)
- IMAGETAG - the image tag (defaults to latest)

To build the image, run the `image` make target, or run the `push` make target to build and push the image:

```bash
make REPOOWNER=${USER} push
```

## Regression testing

A regression test suite utility is built into the image. It requires the pull secrets in `/root/.docker/config.json`, similar to
the `download` command, as it will run `oc-mirror`, but it will avoid downloading the images. New tests should be added in the regression
folder as needed. In addition, developers should run the regression tests prior to posting a pull request.

```console
[root@host core]# time podman run -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:latest -- regression-suite.sh
################################################################################
Wed May 24 18:39:26 UTC 2023: Running test: du-profile
################################################################################
Wed May 24 18:43:54 UTC 2023: Running test: generate-imageset
/usr/local/bin/regression-tests/regression-test-generate-imageset.sh: line 76: [: /mnt/testsuite/imageset.yaml: integer expression expected
################################################################################
Wed May 24 18:43:54 UTC 2023: Running test: invalid-acm-version-format
################################################################################
Wed May 24 18:43:54 UTC 2023: Running test: invalid-mce-version-format
################################################################################
Wed May 24 18:43:54 UTC 2023: Running test: invalid-version-format
################################################################################
Wed May 24 18:43:54 UTC 2023: Running test: keep-stale
Stale file found: /mnt/testsuite/not-a-real-image@sha256_1234567890123456789012345678901234567890123456789012345678901234.tgz
################################################################################
Wed May 24 18:46:48 UTC 2023: Running test: stale-cleanup
################################################################################
Wed May 24 18:49:41 UTC 2023: Running test: unavailable-acm-and-mce-versions
################################################################################
Wed May 24 18:51:39 UTC 2023: Running test: unavailable-acm-version
################################################################################
Wed May 24 18:53:35 UTC 2023: Running test: unavailable-mce-version
################################################################################
Wed May 24 18:54:32 UTC 2023: Running test: unknown-option
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

real    15m7.076s
user    0m0.510s
sys     0m0.383s
[root@host core]#
```
