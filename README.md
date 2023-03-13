# telco-ran-tools #

This repository holds tools used by the Telco RAN team.

- [telco-ran-tools](#telco-ran-tools)
  - [factory-precaching-cli](#factory-precaching-cli)
    - [Motivation](#motivation)
    - [Description](#description)
    - [Building](#building)
    - [Usage](#usage)
    - [Display help](#display-help)

## factory-precaching-cli ##

### Motivation ###

In environments with limited bandwidth and using the RH GitOps ZTP solution to deploy a large number of clusters for RAN
workloads, it might be desirable to avoid the use of the network for downloading all the artifacts required for
bootstrapping and installing OCP.  Remember that the bandwidth to remote DU SNO sites is limited resulting in long
deployment times with the existing ZTP solution. In order to address the bandwidth limitations, a factory pre-staging
solution is required to eliminate the download of artifacts at the remote site.

As artifacts, we refer both to container images and images such as the rootFS which currently can only be pulled from an
HTTP/HTTPS server when booting from the small ISO. One can think of using the RHCOS live full ISO to avoid downloading
the rootFS image, but it is discouraged because there are BMCs that do not accept the use of ISO files with such large
sizes.

This work is targeting hardware where one disk, actually the installation disk, is only available. In this case, when
installing OCP via ZTP/Assisted Installer using the small ISO, we need to:

- Allow copying the rootFS file at boot time from a local disk path instead of using the network. Currently, only HTTP/HTTPS location is allowed via the image-url flag.
- Pre-stage OCP release images in a partition of the installation disk which is saved during OCP installation.
- Pre-stage day-2 operators (specifically telco operators) in the same partition during OCP configuration. In particular when applying the DU profile.

> :warning: The factory-cli tool can target hardware with more than one disk installed as well, however as stated this is not the main objective.

### Description ###

The factory-precaching-cli tool facilitates the pre-staging of servers before they are shipped to the site for later ZTP provisioning.

The tool does the following:

- Creates a partition from the installation disk labelled as data.
- Formats the disk (xfs).
- Creates a GPT data partition at the end of the disk. Size is configurable by the tool.
- Copies the container images required to install OCP.
- Copies the container images required by ZTP to install OCP.
- If requested, day-2 operators are copied to the partition too.
- Downloads the RHCOS rootfs image required by the minimal ISO to boot.

### Building ###

``` console
$ make build
mkdir -p _output || :
go mod tidy && go mod vendor
Running go fmt
go fmt ./...
Running go vet
go vet ./...
# go flags are set in here
./hack/build-binaries.sh
command-line-arguments
```

The binary can be found in _output/factory-precaching-cli

### Usage ###

Detailed information and examples of each stage can be found in the following links:

- [Booting RHCOS live](docs/liveos.md)
- [Partitioning](docs/partitioning.md)
- [Downloading the artifacts](docs/downloading.md)
- [ZTP configuration for precaching](docs/ztp-precaching.md)

### Display help ###

``` console
$ ./factory-precaching-cli --help
factory-precaching-cli is a tool that facilitates pre-caching OpenShift artifacts
in servers to avoid downloading them at provisioning time.

Usage:
  factory-precaching-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  download    Downloads and pre-caches artifacts
  help        Help about any command
  partition   Partitions and formats a disk

Flags:
  -h, --help      help for factory-precaching-cli
  -t, --toggle    Help message for toggle
  -v, --version   version for factory-precaching-cli

Use "factory-precaching-cli [command] --help" for more information about a command.
```
