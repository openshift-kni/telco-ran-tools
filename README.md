# telco-ran-tools

This repository holds tools used by the Telco RAN team.

## factory-precaching-cli

### Description
The factory-precaching-cli tool facilitates the pre-staging of servers before they are shipped to the site for later ZTP provisionig. 

The tool does the following: 
* Formats the disk
* Creates a GPT data partition at the end of the disk. Size is configurable 
* Copies the container images required for Assisted service and bootstrapping to the partition 
* Copies the OCP container images to the partition 
* Release payload
* Cluster Operators 
* Day 2 Operators
* RHCOS rootfs image

### Purpose

Bandwidth to remote DU SNO sites is limited resulting in long deployment times with the existing ZTP solution.

In order to address the bandwidth limitations, a factory pre-staging solution is required to eliminate the download of artifacts at the remote site

### Building
```
[ricky@localhost telco-ran-tools]$ go build -o factory-precaching-cli
```

### Usage

```
# Display help
[ricky@localhost telco-ran-tools]$ ./factory-precaching-cli --help
A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.

Usage:
  factory-prestaging-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  partition   A brief description of your command

Flags:
  -h, --help     help for factory-prestaging-cli
  -t, --toggle   Help message for toggle

Use "factory-prestaging-cli [command] --help" for more information about a command.

# Partition /dev/nvme0n1 device with a 120G size and format it with xfs
[ricky@localhost telco-ran-tools]$ ./factory-precaching-cli partition --device /dev/nvme0n1 --size 120

# Download OpenShift 4.9 release artifacts onto /mnt/data
[ricky@localhost telco-ran-tools]$ ./factory-precaching-cli download -f /mnt/data -r 4.9
```
