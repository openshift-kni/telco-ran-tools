# factory-precaching-cli: Downloading #

- [factory-precaching-cli: Downloading](#factory-precaching-cli-downloading)
  - [Background](#background)
  - [Pre-requisites](#pre-requisites)
  - [Downloading the artifacts](#downloading-the-artifacts)
    - [Preparing for the download](#preparing-for-the-download)
    - [Parallel Downloads](#parallel-downloads)
    - [Precaching an OCP release](#precaching-an-ocp-release)
    - [Precaching the telco 5G RAN operators](#precaching-the-telco-5g-ran-operators)
    - [Custom precaching for disconnected environments](#custom-precaching-for-disconnected-environments)

## Background ##

As mentioned, in order to address the bandwidth limitations, a factory pre-staging solution is required to eliminate the
download of artifacts at the remote site.The factory-precaching-cli (a.k.a factory-precaching-cli) tool facilitates the
pre-staging of servers before they are shipped to the site for later ZTP provisioning. Remember that the idea is to
focus on those servers where only one disk is available and no external disk drive is possible to be attached.

This downloading stage manage both the pull of the OCP release images, and if required, it can also manage the download
of the operators included in the distributed unit (DU) profile for telco 5G RAN sites. More advanced scenarios can be
set up too, such as [downloading operators images from disconnected registries](#custom-precaching-for-disconnected-environments).

> :warning: Notice that the list of operators can vary depending on the OCP version we are about to install. For
  instance, in OCP 4.12 the DU profile adds AMQ, LVM and the baremetal-event (BMER) operators compared to 4.11.

## Pre-requisites ##

- The [partitioning stage](./partitioning.md) must be already executed successfully before moving to the downloading
  stage. In this last step is where all artifacts are stored on the local partition.
- Currently the bare metal server needs to be connected to the Internet to obtain the dependency of OCP release images that need to be pulled down.
- A valid pull secret to the registries involved in the downloading process of the container images is required. At
  least the pull secret that authenticates against the official Red Hat registries is required. It can be obtained from
  the [Red Hat's console UI](https://console.redhat.com/openshift/downloads#tool-pull-secret)
- Enough space in the partition where the artifacts are going to be stored is required. More information can be found on
  [Partitioning pre-requisites](./partitioning.md/#pre-requisites) section.

## Downloading the artifacts ##

In this stage, we can split the tasks up to downloading the OCP release container images and the day-2 operators,
specifically the telco DU profile approved operators. Before starting, however, we need to know in advance what version
of RHACM is going to provision the SNO. The version of the hub cluster will determine what assisted installer container
images are used by the SNO to provision and report back the inventory and progress of the spoke cluster.

You can check the version of ACM and MCE by executing these commands in the hub cluster:

```console
$ oc get csv -A | grep -i advanced-cluster-management
open-cluster-management                            advanced-cluster-management.v2.5.4           Advanced Cluster Management for Kubernetes   2.5.4                 advanced-cluster-management.v2.5.3                Succeeded

oc get csv -A | grep -i multicluster-engine
multicluster-engine                                cluster-group-upgrades-operator.v0.0.3       cluster-group-upgrades-operator              0.0.3                                                                   Pending
multicluster-engine                                multicluster-engine.v2.0.4                   multicluster engine for Kubernetes           2.0.4                 multicluster-engine.v2.0.3                        Succeeded
multicluster-engine                                openshift-gitops-operator.v1.5.7             Red Hat OpenShift GitOps                     1.5.7                 openshift-gitops-operator.v1.5.6-0.1664915551.p   Succeeded
multicluster-engine                                openshift-pipelines-operator-rh.v1.6.4       Red Hat OpenShift Pipelines                  1.6.4                 openshift-pipelines-operator-rh.v1.6.3            Succeeded
```

>:warning: RHACM has as a prereq the MCE, so installing RHACM will also install MCE.

### Preparing for the download ###

Before starting to pull down the images we need to copy a valid pull secret to access the container registry. Notice
that this is done in the server that is going to be installed. It is important to copy the pull secret in the folder
shown below as `config.json`. This is a path where Podman will check by default the credentials to login into any
registry.

```console
mkdir /root/.docker
cp config.json /root/.docker/config.json
```

>:exclamation: The pull secret to access the online Red Hat registries can be found in the [console.redhat.com UI](https://console.redhat.com/openshift/downloads#tool-pull-secret)

It is worth mentioning that if you are using a different registry to pull the content down you need to copy the proper
pull secret. If the local registry uses TLS then you need also to include the certificates from the registry as well.

### Parallel Downloads ###

The factory-precaching-cli tool will use parallel workers to download multiple images simultaneously. The number of
workers to use can be configured by specifying the --parallel (or -p) option, which defaults to 80% of the available
CPUs. Please note that your login shell may be restricted to a subset of CPUs, which will reduce the CPUs available to
the container. If so, it is recommended that you precede your command with `taskset 0xffffffff` to remove this
restriction:

```console
# taskset 0xffffffff podman run --rm quay.io/openshift-kni/telco-ran-tools:1.0 factory-precaching-cli download --help
Downloads and pre-caches artifacts

Usage:
  factory-precaching-cli download [flags]

Flags:
      --acm-version string   Advanced Cluster Management operator version, in X.Y.Z format
  -i, --ai-img strings       Assisted Installer Image(s)
      --du-profile           Pre-cache telco 5G DU operators
  -f, --folder string        Folder to download artifacts
      --generate-imageset    Generate imageset.yaml only
  -h, --help                 help for download
  -a, --img strings          Additional Image(s)
      --mce-version string   MultiCluster Engine operator version, in X.Y.Z format
  -p, --parallel int         Maximum parallel downloads (default 83)
  -r, --release string       OpenShift release version, in X.Y.Z format
  -s, --rm-stale             Remove stale images
  -u, --rootfs-url string    rootFS URL
      --skip-imageset        Skip imageset.yaml generation
  -v, --version              version for download
```

### Precaching an OCP release ###

The factory-precaching-cli tool allows us to precache all the container images required to provision a specific OCP release. In the following execution we are:

- Precaching 4.11.5 OCP release
- Copying all the dependent artifacts into /mnt
- Mounting the pull secret that we just created into /root/.docker
- Including an extra image that we want to be copied and extracted during the installation stage (--img)

Note that we could also specify explicit installer images to be precached, using the --ai-img option.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools -- \
   factory-precaching-cli download -r 4.11.5 --acm-version 2.5.4 --mce-version 2.0.4 -f /mnt \
   --img quay.io/alosadag/troubleshoot

Generated /mnt/imageset.yaml
Generating list of pre-cached artifacts...

Queueing 176 of 176 images for download, with 83 workers.

Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:f68c0e6f5e17b0b0f7ab2d4c39559ea89f900751e64b97cb42311a478338d9c3
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:7753a8d9dd5974be8c90649aadd7c914a3d8a1f1e016774c7ac7c9422e9f9958
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:370e47a14c798ca3f8707a38b28cfc28114f492bb35fe1112e55d1eb51022c99
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:e8f55ebd974f99b7056e1fa308d9abacfa285758e9ee055f8ed8438f410f1325
...
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:edec37e7cd8b1611d0031d45e7958361c65e2005f145b471a8108f1b54316c07
Downloaded artifact [1/176]: ocp-v4.0-art-dev@sha256_4f04793bd109ecba2dfe43be93dc990ac5299272482c150bd5f2eee0f80c983b
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:94ade3f187a4c32597c7f1f1a3bbea6849f4c31dc28e61e05fc0a5c303e8fecd
Downloaded artifact [2/176]: ocp-v4.0-art-dev@sha256_5a862d169b7093d18235bf6029bc03ab298a1af2806224fb50771ee7c7a0b82e
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:f48b68d5960ba903a0d018a10544ae08db5802e21c2fa5615a14fc58b1c1657c
Downloaded artifact [3/176]: ocp-v4.0-art-dev@sha256_a190cc6dfb86080ff1c31ba717635f5a948b0b75772cd5ba10ffa09dcd6e6daa
...
Downloaded artifact [176/176]: ocp-v4.0-art-dev@sha256_e68d705c63061c735fb00f0d2fec361d2d6dfa854a8cc05a92753c08fbaf4684

Summary:

Release:                            4.11.5
ACM Version:                        2.5.4
MCE Version:                        2.0.4
Include DU Profile:                 No
Workers:                            83

Total Images:                       176
Downloaded:                         176
Skipped (Previously Downloaded):    0
Download Failures:                  0
Time for Download:                  4m16s
```

Verify that all the images are compressed in the target folder (it is suggested to be /mnt) of the bare metal server:

```console
$ ls -l /mnt
-rw-r--r--. 1 root root  136352323 Oct 31 15:19 ocp-v4.0-art-dev@sha256_edec37e7cd8b1611d0031d45e7958361c65e2005f145b471a8108f1b54316c07.tgz
-rw-r--r--. 1 root root  156092894 Oct 31 15:33 ocp-v4.0-art-dev@sha256_ee51b062b9c3c9f4fe77bd5b3cc9a3b12355d040119a1434425a824f137c61a9.tgz
-rw-r--r--. 1 root root  172297800 Oct 31 15:29 ocp-v4.0-art-dev@sha256_ef23d9057c367a36e4a5c4877d23ee097a731e1186ed28a26c8d21501cd82718.tgz
-rw-r--r--. 1 root root  171539614 Oct 31 15:23 ocp-v4.0-art-dev@sha256_f0497bb63ef6834a619d4208be9da459510df697596b891c0c633da144dbb025.tgz
-rw-r--r--. 1 root root  160399150 Oct 31 15:20 ocp-v4.0-art-dev@sha256_f0c339da117cde44c9aae8d0bd054bceb6f19fdb191928f6912a703182330ac2.tgz
-rw-r--r--. 1 root root  175962005 Oct 31 15:17 ocp-v4.0-art-dev@sha256_f19dd2e80fb41ef31d62bb8c08b339c50d193fdb10fc39cc15b353cbbfeb9b24.tgz
-rw-r--r--. 1 root root  174942008 Oct 31 15:33 ocp-v4.0-art-dev@sha256_f1dbb81fa1aa724e96dd2b296b855ff52a565fbef003d08030d63590ae6454df.tgz
-rw-r--r--. 1 root root  246693315 Oct 31 15:31 ocp-v4.0-art-dev@sha256_f44dcf2c94e4fd843cbbf9b11128df2ba856cd813786e42e3da1fdfb0f6ddd01.tgz
-rw-r--r--. 1 root root  170148293 Oct 31 15:00 ocp-v4.0-art-dev@sha256_f48b68d5960ba903a0d018a10544ae08db5802e21c2fa5615a14fc58b1c1657c.tgz
-rw-r--r--. 1 root root  168899617 Oct 31 15:16 ocp-v4.0-art-dev@sha256_f5099b0989120a8d08a963601214b5c5cb23417a707a8624b7eb52ab788a7f75.tgz
-rw-r--r--. 1 root root  176592362 Oct 31 15:05 ocp-v4.0-art-dev@sha256_f68c0e6f5e17b0b0f7ab2d4c39559ea89f900751e64b97cb42311a478338d9c3.tgz
-rw-r--r--. 1 root root  157937478 Oct 31 15:37 ocp-v4.0-art-dev@sha256_f7ba33a6a9db9cfc4b0ab0f368569e19b9fa08f4c01a0d5f6a243d61ab781bd8.tgz
-rw-r--r--. 1 root root  145535253 Oct 31 15:26 ocp-v4.0-art-dev@sha256_f8f098911d670287826e9499806553f7a1dd3e2b5332abbec740008c36e84de5.tgz
-rw-r--r--. 1 root root  158048761 Oct 31 15:40 ocp-v4.0-art-dev@sha256_f914228ddbb99120986262168a705903a9f49724ffa958bb4bf12b2ec1d7fb47.tgz
-rw-r--r--. 1 root root  167914526 Oct 31 15:37 ocp-v4.0-art-dev@sha256_fa3ca9401c7a9efda0502240aeb8d3ae2d239d38890454f17fe5158b62305010.tgz
-rw-r--r--. 1 root root  164432422 Oct 31 15:24 ocp-v4.0-art-dev@sha256_fc4783b446c70df30b3120685254b40ce13ba6a2b0bf8fb1645f116cf6a392f1.tgz
-rw-r--r--. 1 root root  306643814 Oct 31 15:11 troubleshoot@sha256_b86b8aea29a818a9c22944fd18243fa0347c7a2bf1ad8864113ff2bb2d8e0726.tgz
```

### Precaching the telco 5G RAN operators ###

Aside from precaching the OCP release, we can also precache the day-2 operators used in telco 5G RAN. They are known as
well as the telco 5G RAN distributed unit (DU) profile. They depend on the version of OCP that is going to be installed.
However, you just need to add the `--du-profile` argument so that the factory-precaching-cli will do the hard work for
you.

Notice that you need also to include the ACM and MCE versions, using `--acm-version` and `--mce-version`, so that the
tool figures out what containers images from RHACM and MCE operators need to pre-stage. Note the ACM version is only
required if `--du-profile` is specified, while MCE version is always needed.

Please also note the `--hub-version` option has been deprecated in favour of the separate `--acm-version` and
`--mce-version` options, to support independent versioning.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli \
    download -r 4.11.5 --acm-version 2.5.4 --mce-version 2.0.4 -f /mnt \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \

Generated /mnt/imageset.yaml
Generating list of pre-cached artifacts...

Queueing 376 of 376 images for download, with 83 workers.

Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c7cf9437d2b8aae03f49793f4f3f3684c8948c849f2838d53ad8472dadc62677
Downloading: registry.redhat.io/rhacm2/acm-volsync-addon-controller-rhel8@sha256:548c1b6121c3f0d0269bf9347a25c83bf7ed98360b0828f0a3faea0d7ded05d0
Downloading: registry.redhat.io/rhacm2/insights-client-rhel8@sha256:a49d0c97e5daae7835e935f5d10f937c05d89edd29c452a675d34789d72849c0
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:a5267b46513453509e4d189c2621a2f578f216e85d78e22f885e8c358f954dd3
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:7456e7a6c1c4d1161a2ff1df57228a86d56c2cf6054d4ea65ad6b45404088293
...
Downloading: registry.redhat.io/multicluster-engine/cluster-proxy-rhel8@sha256:e07eaed7ce86a9a72d0ef617941bcb2339c27e206ae477081616223f39bfa22d
Downloaded artifact [1/376]: cluster-logging-operator-bundle@sha256_4e6ada19c48d471db0513a1b5acba91ebecca42ce5127778b96a72d62af85289
Downloading: registry.connect.redhat.com/intel/sriov-fec-operator-bundle@sha256:bee33416cfe3b7dd9df571e5c448ef431bbf4802d182b30f8acae775a995f0d3
Downloaded artifact [2/376]: acm-operator-bundle@sha256_12a06b081a8cdea335f7388112994ef912e18a54110da80fb56c728164666609
Downloading: registry.redhat.io/multicluster-engine/cluster-api-rhel8@sha256:b5c042364770d729a57412e9db8f1049efc8b1a63430ae3c296fbdd0519672fd
Downloaded artifact [3/376]: sriov-fec-operator-bundle@sha256_bee33416cfe3b7dd9df571e5c448ef431bbf4802d182b30f8acae775a995f0d3
...
Downloaded artifact [375/376]: ocp-v4.0-art-dev@sha256_0518bb71854ae899c5601f44ae9525a8e7a502cd8114081661e9216c1651a130
Downloaded artifact [376/376]: hive-rhel8@sha256_609cddbaacc9f50906119104da8f5323c0630cb895bd8829ffd719e76c50ef50

Summary:

Release:                            4.11.5
ACM Version:                        2.5.4
MCE Version:                        2.0.4
Include DU Profile:                 Yes
Workers:                            83

Total Images:                       376
Downloaded:                         376
Skipped (Previously Downloaded):    0
Download Failures:                  0
Time for Download:                  9m43s
```

>:exclamation: Notice that the number of containers precached highly increases because of the operators included in the DU profile. In the previous example we moved from 176 container images to 376.

### Custom precaching for disconnected environments ###

By default the factory-precaching-cli tool enables the argument `--generate-imageset`, which will create an `imageset`
yaml definition including the OCP release and the optional telco 5G RAN operators required for the specific OCP release.
However, this automatically generated file can be modified to our needs. In the following example, we are going to
generate an `ImageSetConfiguration` based on the arguments passed to the tool and then stop.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli \
    download -r 4.11.5 --acm-version 2.5.4 --mce-version 2.0.4 -f /mnt \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \
    --generate-imageset

Generated /mnt/imageset.yaml
...
```

Based on the options included in the call, an imageset like the following one is created. Notice that an `ImageSetConfiguration` is a custom resource definition (CRD) managed by oc-mirror. In this CR you can see that:

- The OCP release version and channel match the one passed to the tool.
- Additional images are included too,
- The operator's section includes the 5G RAN DU operators for the 4.11.z release of OpenShift: LSO, PTP, SR-IOV, Logging and the Accelerator operator.
- The RHACM and MCE operators match the `--acm-version` and `--mce-version` versions passed to the tool.

```yaml
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
mirror:
  platform:
    channels:
    - name: stable-4.11
      minVersion: 4.11.5
      maxVersion: 4.11.5
  additionalImages:
    - name: quay.io/alosadag/troubleshoot
  operators:
    - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.11
      packages:
        - name: advanced-cluster-management
          channels:
             - name: 'release-2.6'
             - name: 'release-2.5'
               minVersion: 2.5.4
               maxVersion: 2.5.4
        - name: multicluster-engine
          channels:
             - name: 'stable-2.1'
             - name: 'stable-2.0'
               minVersion: 2.0.4
               maxVersion: 2.0.4        
        - name: local-storage-operator
          channels:
            - name: 'stable'
        - name: ptp-operator
          channels:
            - name: 'stable'
        - name: sriov-network-operator
          channels:
            - name: 'stable'
        - name: cluster-logging
          channels:
            - name: 'stable'
    - catalog: registry.redhat.io/redhat/certified-operator-index:v4.11
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
```

At this point, we can just modify the imageset definition to include new operators or additional images. On the other
hand, we can just remove some of them if they are not going to be used. Another interesting idea is that we can replace
operators or catalog sources to use the ones that are in a local or disconnected registry instead of the official Red
Hat registry. That's what we are going to do:

```yaml
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
mirror:
  platform:
...
  operators:
    - catalog: eko4.cloud.lab.eng.bos.redhat.com:8443/redhat/redhat-operator-index:v4.11
      packages:
        - name: advanced-cluster-management
          channels:
             - name: 'release-2.6'
             - name: 'release-2.5'
               minVersion: 2.5.4
               maxVersion: 2.5.4
        - name: multicluster-engine
          channels:
             - name: 'stable-2.1'
             - name: 'stable-2.0'
               minVersion: 2.0.4
               maxVersion: 2.0.4        
        - name: local-storage-operator
          channels:
            - name: 'stable'
        - name: ptp-operator
          channels:
            - name: 'stable'
        - name: sriov-network-operator
          channels:
            - name: 'stable'
        - name: cluster-logging
          channels:
            - name: 'stable'
    - catalog: eko4.cloud.lab.eng.bos.redhat.com:8443/redhat/certified-operator-index:v4.11
...
```

Then, we need to start the downloading of the images by explicitly (--skip-imageset) asking the tool not to generate a new `imageSetConfiguration`:

>:warning: If you are going to pull content from a different registry you have to include the proper pull secret in the .docker/config.json file. Also, you have to probably include the proper certificates too.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:1.0 -- factory-precaching-cli \
    download -r 4.11.5 --acm-version 2.5.4 --mce-version 2.0.4 -f /mnt \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \
    --skip-imageset

Generating list of pre-cached artifacts...
error: unable to run command oc-mirror -c /mnt/imageset.yaml file:///tmp/fp-cli-3218002584/mirror --ignore-history --dry-run: Creating directory: /tmp/fp-cli-3218002584/mirror/oc-mirror-workspace/src/publish
Creating directory: /tmp/fp-cli-3218002584/mirror/oc-mirror-workspace/src/v2
Creating directory: /tmp/fp-cli-3218002584/mirror/oc-mirror-workspace/src/charts
Creating directory: /tmp/fp-cli-3218002584/mirror/oc-mirror-workspace/src/release-signatures
backend is not configured in /mnt/imageset.yaml, using stateless mode
backend is not configured in /mnt/imageset.yaml, using stateless mode
No metadata detected, creating new workspace
level=info msg=trying next host error=failed to do request: Head "https://eko4.cloud.lab.eng.bos.redhat.com:8443/v2/redhat/redhat-operator-index/manifests/v4.11": x509: certificate signed by unknown authority host=eko4.cloud.lab.eng.bos.redhat.com:8443

The rendered catalog is invalid.

Run "oc-mirror list operators --catalog CATALOG-NAME --package PACKAGE-NAME" for more information.

error: error rendering new refs: render reference "eko4.cloud.lab.eng.bos.redhat.com:8443/redhat/redhat-operator-index:v4.11": error resolving name : failed to do request: Head "https://eko4.cloud.lab.eng.bos.redhat.com:8443/v2/redhat/redhat-operator-index/manifests/v4.11": x509: certificate signed by unknown authority
```

The previous error is basically saying that we are missing the certificates of the new registry we want to pull content from. In order to solve this common issue, we have to copy the registry certificate into our server and update the certificates trust store:

>:exclamation: Remember that our server is currently running a live ISO RHCOS image.

``` console
# cp /tmp/eko4-ca.crt /etc/pki/ca-trust/source/anchors/.
# update-ca-trust 
```

Next, we just need to mount the host `/etc/pki` folder into the factory-precaching-cli container image. The
factory-precaching-cli image is built on a UBI RHEL image, so paths and locations for certificates are going hand by
hand with RHCOS (based on RHEL too). Take that into account when mounting host folders.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker -v /etc/pki:/etc/pki --privileged --rm quay.io/openshift-kni/telco-ran-tools:1.0 -- \
    factory-precaching-cli download -r 4.11.5 --acm-version 2.5.4 --mce-version 2.0.4 -f /mnt \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \
    --skip-imageset
 
Generating list of pre-cached artifacts...
...
```
