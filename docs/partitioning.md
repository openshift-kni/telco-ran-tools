# Factory-cli: Partitioning #

- [Factory-cli: Partitioning](#factory-cli-partitioning)
  - [Background](#background)
  - [Pre-requisites](#pre-requisites)
  - [Partitioning and formatting](#partitioning-and-formatting)
    - [Verify the disk is cleared](#verify-the-disk-is-cleared)
    - [Create the partition](#create-the-partition)
    - [Mount the partition](#mount-the-partition)

## Background ##

As mentioned, the idea is to focus on those servers where only one disk is available and no external disk drive is
possible to be attached. Notice that assisted installer, which is part of the ZTP flow, leverages the coreos-installer
utility to write RHCOS to disk. Therefore, if we boot from a pre-installed RHCOS on the device, the utility will
complain because the device is in use and cannot finish the process of writing. Then, the only way we have to run the
full pre-caching process is by booting from a live ISO and using the factory-cli tool from a container image to
partition and pre-cache all the artifacts required.

> :warning: RHCOS requires the disk to not be in use when is about to be written by an RHCOS image. Reinstalling onto the current boot disk is an unusual requirement, and the coreos-installer utility wasn't designed for it.

## Pre-requisites ##

As a requirement before starting the partitioning process with the factory-cli, we need the disk to be not partitioned.
If it is partitioned we can boot from an [RHCOS live ISO](https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/latest/rhcos-4.11.9-x86_64-live.x86_64.iso),
delete the partition and wipe the full device. See the process below for deleting a partition of the nvme0n1 disk:

``` console
[root@snonode /]# lsblk
NAME        MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
loop0         7:0    0  93.8G  0 loop /run/ephemeral
loop1         7:1    0 897.3M  1 loop /sysroot
sr0          11:0    1   999M  0 rom  /run/media/iso
nvme0n1     259:1    0   1.5T  0 disk 
└─nvme0n1p1 259:3    0   300G  0 part 

[root@snonode /]# fdisk /dev/nvme0n1

Welcome to fdisk (util-linux 2.32.1).
Changes will remain in memory only, until you decide to write them.
Be careful before using the write command.


Command (m for help): d
Selected partition 1
Partition 1 has been deleted.

Command (m for help): w
The partition table has been altered.
Calling ioctl() to re-read partition table.
Syncing disks.

[root@snonode /]# wipefs -a /dev/nvme0n1
/dev/nvme0n1: 8 bytes were erased at offset 0x00000200 (gpt): 45 46 49 20 50 41 52 54
/dev/nvme0n1: 8 bytes were erased at offset 0x1749a955e00 (gpt): 45 46 49 20 50 41 52 54
/dev/nvme0n1: 2 bytes were erased at offset 0x000001fe (PMBR): 55 aa
/dev/nvme0n1: calling ioctl to re-read partition table: Success

[root@snonode /]# lsblk
NAME    MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
loop0     7:0    0  93.8G  0 loop /run/ephemeral
loop1     7:1    0 897.3M  1 loop /sysroot
sr0      11:0    1   999M  0 rom  /run/media/iso
nvme0n1 259:1    0   1.5T  0 disk 
```

> :exclamation: It is recommended to use NVME disk in your servers.

A second prerequisite is being able to pull the quay.io/openshift-kni/telco-ran-tools:1.0 that will be used to run
the factory-cli tool. The image is publicly available in quay.io but if you are in a disconnected environment or have a
corporate private registry, you will need to copy the image there so it can be downloaded to the server.

``` console
[root@snonode /]# podman pull quay.io/openshift-kni/telco-ran-tools:1.0
Trying to pull quay.io/openshift-kni/telco-ran-tools:1.0...
Getting image source signatures
Copying blob 97da74cc6d8f done  
Copying blob 8700ab5d88a1 done  
Copying blob d8190195889e done  
Copying blob b250919afbec done  
Copying blob 5f6dcacfa818 done  
Copying blob 6b1b76a0904e done  
Copying blob cf95c38e62d8 done  
Copying blob 86cf6373f6c4 done  
Copying config 4b99f87ac7 done  
Writing manifest to image destination
Storing signatures
4b99f87ac7872f8b6cdfea50ec0aed44d5145a33b171e22c33b866ce15759c72
```

``` console
[root@snonode /]# podman run quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli -v
factory-precaching-cli version 20221018.120852+main.feecf17
```

Finally, we need to be sure that the disk is big enough to install OpenShift and also precache all the container images required: OCP release and day2 operators. Based on our experience:

- If you are going to precache only OCP release artifacts a 100GB partition is enough
- If you want to precache both OCP release artifacts and the DU profile day2 operators, around 250GB of disk space is required.
- Lastly, take into account that OCP installation requires a minimum 120GB to be installed. So, you need to add this size to precached partition.

> :warning: The factory-cli tool allows you to precache any other container image you will need for your environment. Then, make sure that the size of those extra images is taken into account in the sizing exercise.

## Partitioning and formatting ##

Let's start with the partitioning, for that we will use the partition argument from the factory-cli. See here all the options we have:

``` console
[root@snonode /]# podman run quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli partition --help
Partitions and formats a disk

Usage:
  factory-precaching-cli partition [flags]

Flags:
  -d, --device string   Device to be partitioned
  -h, --help            help for partition
  -s, --size int        Partition size in GB (default 100)
  -v, --version         version for partition
```

### Verify the disk is cleared ###

As mentioned in the previous section, we need our disk to be empty. Please make sure it is before starting the process. This is an example of a nvme0n1 device with no partitions:

``` console
[root@snonode /]# lsblk
NAME    MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
loop0     7:0    0  93.8G  0 loop /run/ephemeral
loop1     7:1    0 897.3M  1 loop /sysroot
sr0      11:0    1   999M  0 rom  /run/media/iso
nvme0n1 259:1    0   1.5T  0 disk 
```

Also, it is suggested to erase filesystem, raid or partition-table signatures from the device:

``` console
[root@snonode /]# wipefs -a /dev/nvme0n1
/dev/nvme0n1: 8 bytes were erased at offset 0x00000200 (gpt): 45 46 49 20 50 41 52 54
/dev/nvme0n1: 8 bytes were erased at offset 0x1749a955e00 (gpt): 45 46 49 20 50 41 52 54
/dev/nvme0n1: 2 bytes were erased at offset 0x000001fe (PMBR): 55 aa
```

> :warning: The tool will fail if the disk is not empty, as it uses the partition number 1 of the device for precaching the artifacts.

### Create the partition ###

Now that we have a device ready, we are going to create a single partition and a GPT partition table. This partition is going to be automatically labeled as “data” and created at the end of the device otherwise the partition will be overridden by the coreos-installer.

> :exclamation: The coreos-installer requires the partition to be created at the end of the device and to be labeled as
  "data". Both requirements are necessary to save the partition when writing the RHCOS image to disk.

Notice that the container must run as privileged since we are formatting host devices. Also, mount the /dev folder so
that the process can be executed inside the container. Finally, notice that the size of the partition is 250 GB since we
expect to precache also the day-2 operators required for the DU profile.

``` console
[root@snonode /]# podman run -v /dev:/dev --privileged -it --rm  quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli partition -d /dev/nvme0n1 -s 250
Partition /dev/nvme0n1p1 is being formatted

[root@snonode /]# lsblk
NAME        MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
loop0         7:0    0  93.8G  0 loop /run/ephemeral
loop1         7:1    0 897.3M  1 loop /sysroot
sr0          11:0    1   999M  0 rom  /run/media/iso
nvme0n1     259:1    0   1.5T  0 disk 
└─nvme0n1p1 259:3    0   250G  0 part 
```

In order to verify the proper status of the partition we can query using the gdisk tool included with RHCOS. See that:

- The device now has a GPT partition table
- The partition is actually using the latest sectors of the device.
- The partition has been correctly labeled as 'data'

``` console
[root@snonode /]# gdisk -l /dev/nvme0n1
GPT fdisk (gdisk) version 1.0.3

Partition table scan:
  MBR: protective
  BSD: not present
  APM: not present
  GPT: present

Found valid GPT with protective MBR; using GPT.
Disk /dev/nvme0n1: 3125627568 sectors, 1.5 TiB
Model: Dell Express Flash PM1725b 1.6TB SFF    
Sector size (logical/physical): 512/512 bytes
Disk identifier (GUID): CB5A9D44-9B3C-4174-A5C1-C64957910B61
Partition table holds up to 128 entries
Main partition table begins at sector 2 and ends at sector 33
First usable sector is 34, last usable sector is 3125627534
Partitions will be aligned on 2048-sector boundaries
Total free space is 2601338846 sectors (1.2 TiB)

Number  Start (sector)    End (sector)  Size       Code  Name
   1      2601338880      3125627534   250.0 GiB   8300  data
```

### Mount the partition ###

Now, that we have verified the partition was created as expected. We can mount the device into /mnt.

> :warning: It is highly suggested to mount the device into /mnt since the mounting point is taken into account during ZTP preparation stage.

Notice that the partition is also formatted as xfs.

``` console
[root@snonode /]# lsblk -f /dev/nvme0n1
NAME        FSTYPE LABEL UUID                                 MOUNTPOINT
nvme0n1                                                       
└─nvme0n1p1 xfs          1bee8ea4-d6cf-4339-b690-a76594794071 
```

Then execute the mount in the terminal, there is no need to add the mount point in the /etc/fstab since we are running a live ISO.

``` console
[root@snonode /]# mount /dev/nvme0n1p1 /mnt/

[root@snonode /]# lsblk
NAME        MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
loop0         7:0    0  93.8G  0 loop /run/ephemeral
loop1         7:1    0 897.3M  1 loop /sysroot
sr0          11:0    1   999M  0 rom  /run/media/iso
nvme0n1     259:1    0   1.5T  0 disk 
└─nvme0n1p1 259:2    0   250G  0 part /var/mnt
```

> :exclamation: Notice that the mount point is actually /var/mnt. This is because the /mnt folder in RHCOS is basically a link to /var/mnt.
