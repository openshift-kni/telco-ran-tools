# factory-precaching-cli: Downloading #

## Background ##

As mentioned, in order to address the bandwidth limitations, a factory pre-staging solution is required to eliminate the download of artifacts at the remote site.The factory-precaching-cli (a.k.a factory-precaching-cli) tool facilitates the pre-staging of servers before they are shipped to the site for later ZTP provisioning. Remember that the idea is to focus on those servers where only one disk is available and no external disk drive is possible to be attached.

This downloading stage manage both the pull of the OCP release images, and if required, it can also manage the download of the operators included in the distributed unit (DU) profile for telco 5G RAN sites. More advanced scenarios can be set up too, such as [downloading operators images from disconnected registries](#custom-precaching-for-disconnected-environments).

> :warning: Notice that the list of operators can vary depending on the OCP version we are about to install. For instance, in OCP 4.12 the DU profile adds AMQ, LVM and the baremetal-event (BMER) operators compared to 4.11.

## Pre-requisites

* The [partitioning stage](./partitioning.md) must be already executed successfully before moving to the downloading stage. In this last step is where all artifacts are stored on the local partition.
* Currently the bare metal server needs to be connected to the Internet to obtain the dependency of OCP release images that need to be pulled down.
* A valid pull secret to the registries involved in the downloading process of the container images is required. At least the pull secret that authenticates against the official Red Hat registries is required. It can be obtained from the [Red Hat's console UI](https://console.redhat.com/openshift/downloads#tool-pull-secret)
* Enough space in the partition where the artifacts are going to be stored is required. More information can be found on [Partitioning pre-requisites](./partitioning.md/#pre-requisites) section.


 ## Downloading the artifacts

In this stage, we can split the tasks up to downloading the OCP release container images and the day-2 operators, specifically the telco DU profile approved operators. Before starting, however, we need to know in advance what version of RHACM is going to provision the SNO. The version of the hub cluster will determine what assisted installer container images are used by the SNO to provision and report back the inventory and progress of the spoke cluster.

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

>:warning: RHACM has as a prereq the MCE, so installing RHACM will also install MCE. Said that, currently, a decision was made that a given version of ACM would only work with a corresponding version of MCE. Those pairings are or will be: ACM 2.5 with MCE 2.0, ACM 2.6 with MCE 2.1 and ACM 2.7 (will be) with MCE 2.2.

### Obtaining the assisted-installer images

The assisted installer container images are being used during the discovery stage of ZTP. In order to precache them, we need to know what versions are going to be used by our hub cluster. We can obtain that information by just querying the `assisted-service` configMap:

>:exclamation: Notice that the namespace might be different depending on the RHACM version installed on the hub cluster.

```console
$ oc get cm assisted-service -n open-cluster-management -oyaml | grep -E "AGENT_DOCKER_IMAGE|CONTROLLER_IMAGE|INSTALLER_IMAGE"
  AGENT_DOCKER_IMAGE: registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306
  CONTROLLER_IMAGE: registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd
  INSTALLER_IMAGE: registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027
```

Then save this information because it is being to be used for downloading the artifacts.

### Preparing for the download

Before starting to pull down the images we need to copy a valid pull secret to access the container registry. Notice that this is done in the server that is going to be installed. It is important to copy the pull secret in the folder shown below as `config.json`. This is a path where Podman will check by default the credentials to login into any registry.

```console
$ mkdir /root/.docker
$ cp config.json /root/.docker/config.json
```

>:exclamation: The pull secret to access the online Red Hat registries can be found in the [console.redhat.com UI](https://console.redhat.com/openshift/downloads#tool-pull-secret)

It is worth mentioning that if you are using a different registry to pull the content down you need to copy the proper pull secret. If the local registry uses TLS then you need also to include the certificates from the registry as well.

### Parallel Downloads

The factory-precaching-cli tool will use parallel workers to download multiple images simultaneously. The number of workers to use can be configured by specifying the --parallel (or -p) option, which defaults to 80% of the available CPUs. Please note that your login shell may be restricted to a subset of CPUs, which will reduce the CPUs available to the container. If so, it is recommended that you precede your command with `taskset 0xffffffff` to remove this restriction:

```console
# taskset 0xffffffff podman run --rm quay.io/openshift-kni/telco-ran-tools:latest factory-precaching-cli download --help
Downloads and pre-caches artifacts

Usage:
  factory-precaching-cli download [flags]

Flags:
  -i, --ai-img strings       Assisted Installer Image(s)
      --du-profile           Pre-cache telco 5G DU operators
  -f, --folder string        Folder to download artifacts
      --generate-imageset    Generate imageset.yaml only
  -h, --help                 help for download
      --hub-version string   RHACM operator version in a.x.z format
  -a, --img strings          Additional Image(s)
  -p, --parallel int         Maximum parallel downloads (default 83)
  -r, --release string       OpenShift release version
  -s, --rm-stale             Remove stale images
  -u, --rootfs-url string    rootFS URL
      --skip-imageset        Skip imageset.yaml generation
  -v, --version              version for download
```

### Precaching an OCP release

The factory-precaching-cli tool allows us to precache all the container images required to provision a specific OCP release. In the following execution we are:

* Precaching 4.11.5 OCP release
* Copying all the dependent artifacts into /mnt
* Mounting the pull secret that we just created into /root/.docker
* Including the assisted installer images that we queried before into the --ai-img list
* Including an extra image that we want to be copied and extracted during the installation stage (--img)

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools -- \
   factory-precaching-cli download -r 4.11.5 --hub-version 2.5.4 -f /mnt \
   --ai-img registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306 \
   --ai-img registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd \
   --ai-img registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027 \
   --img quay.io/alosadag/troubleshoot

Generated /mnt/imageset.yaml
Generating list of pre-cached artifacts...

Queueing 175 of 175 images for download, with 83 workers.

Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:370e47a14c798ca3f8707a38b28cfc28114f492bb35fe1112e55d1eb51022c99
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:2b3dae7d4858ea5b4db589a04d5d30c50177085a3a7fa768e3507f7c3c3fb120
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:ee51b062b9c3c9f4fe77bd5b3cc9a3b12355d040119a1434425a824f137c61a9
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:535f49d0147cbbfbdd3759b09eeef8968ea3de9dc1e2156721726566abcaeb57
...
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:b008d8526ffdf997d62309ec12e7b6ba673f76235a5ffe7236db57f407dc24ba
Downloaded artifact [1/175]: ocp-v4.0-art-dev@sha256_174b4f8995157d0f1b5533c9d179c1eb681415a0bd092b4c1b14c2ed1f28083c
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c7cf9437d2b8aae03f49793f4f3f3684c8948c849f2838d53ad8472dadc62677
Downloaded artifact [2/175]: ocp-v4.0-art-dev@sha256_d44c355a799955de2bdf34a598673ecbadd1c25505057a9d7449b1d11a3d6ec4
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:39d850d6d18d2e00d9501ffec0836f911619e3e2fdc57a01ca3ae6992e6667db
Downloaded artifact [3/175]: ocp-v4.0-art-dev@sha256_c5ba93680ec2c9e0ca419424fa6b2416fa542287420499dc655f03b968564013
...
Downloaded artifact [175/175]: ocp-v4.0-art-dev@sha256_e68d705c63061c735fb00f0d2fec361d2d6dfa854a8cc05a92753c08fbaf4684
175 images of 175 downloaded, with 83 workers, in: 4m10.580718762s
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

### Precaching the telco 5G RAN operators

Aside from precaching the OCP release, we can also precache the day-2 operators used in telco 5G RAN. They are known as well as the telco 5G RAN distributed unit (DU) profile. They depend on the version of OCP that is going to be installed. However, you just need to add the `--du-profile` argument so that the factory-precaching-cli will do the hard work for you. 

Notice that you need also to include the ACM hub version, using `--hub-version`, so that the tool figures out what containers images from RHACM and MCE operators need to pre-stage.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:latest -- factory-precaching-cli \
    download -r 4.11.5 --hub-version 2.5.4 -f /mnt \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306 \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027 \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \

Generated /mnt/imageset.yaml
Generating list of pre-cached artifacts...

Queueing 375 of 375 images for download, with 83 workers.

Downloading: registry.connect.redhat.com/intel/sriov-fec-daemon@sha256:d7b91d4bf4e57415f9a9ca46619d98bfafe5e2d7c5e8072dedd2bcd638725c8d
Downloading: registry.redhat.io/openshift4/ose-ptp@sha256:7795151ae4b3369a6d8a5c14e16b3fbbcba58bb4c852c542d2471fbb4fab7713
Downloading: registry.redhat.io/multicluster-engine/agent-service-rhel8@sha256:4d43dbb43242d98b7f1ed60dbd8613c909a3f9ac86c6ebd5e74b2c47c5d733e9
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:b755466cb222868097e77b8450f95ee8c04b62687201fbf812ef7b5db384729b
Downloading: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:e8f55ebd974f99b7056e1fa308d9abacfa285758e9ee055f8ed8438f410f1325
Downloading: registry.redhat.io/openshift-logging/log-file-metric-exporter-rhel8@sha256:4e850fced9cc85ff23cb60ec8628f581b32f36313714a205c67c9a088a1b4f74
...
Downloading: registry.redhat.io/multicluster-engine/clusterclaims-controller-rhel8@sha256:6d1333b2db47a9c299e63494962e1a5bb109c143a1296acf7145303d61211332
Downloaded artifact [1/375]: ose-local-storage-operator-bundle@sha256_341ccfb41fb53eda21a940fb69d814a8927ad8c81b91b2544f5ddac311b07602
Downloaded artifact [2/375]: ocp-v4.0-art-dev@sha256_e9606f66904be6a38dfa0f80fca91b83bb672e61ad42aec1dd28f5c06a281e2c
Downloading: registry.redhat.io/openshift-logging/cluster-logging-operator-bundle@sha256:a78fd59207ef6cc8ddaaa3f3ae7140b7678e96e0677517623d1537afc05e84dd
Downloading: registry.redhat.io/rhacm2/endpoint-monitoring-rhel8-operator@sha256:4f7e4f762cba270cd9aa731371a66b16f1c19792f14cd5046f138d1c0f80b36c
Downloaded artifact [3/375]: sriov-fec-operator@sha256_5962e20d88f031d5b1b9c726120b14b39ef78b42d9f82322554de57129015412
...
Downloaded artifact [374/375]: fluentd-rhel8@sha256_842077788b4434800127d63b4cd5d8cfaa1cfd3ca1dfd8439de30c6e8ebda884
Downloaded artifact [375/375]: ocp-v4.0-art-dev@sha256_e68d705c63061c735fb00f0d2fec361d2d6dfa854a8cc05a92753c08fbaf4684
375 images of 375 downloaded, with 83 workers, in: 9m57.816237255s
```

>:exclamation: Notice that the number of containers precached highly increases because of the operators included in the DU profile. In the previous example we moved from 176 container images to 379.


### Custom precaching for disconnected environments

By default the factory-precaching-cli tool enables the argument `--generate-imageset`, which will create an `imageset` yaml definition including the OCP release and the optional telco 5G RAN operators required for the specific OCP release. However, this automatically generated file can be modified to our needs. In the following example, we are going to generate an `ImageSetConfiguration` based on the arguments passed to the tool and then stop.


```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:latest -- factory-precaching-cli \
    download -r 4.11.5 --hub-version 2.5.4 -f /mnt \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306 \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027 \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \
    --generate-imageset

Generated /mnt/imageset.yaml
...
```

Based on the options included in the call, an imageset like the following one is created. Notice that an `ImageSetConfiguration` is a custom resource definition (CRD) managed by oc-mirror. In this CR you can see that:

* The OCP release version and channel match the one passed to the tool.
* Additional images are included too,
* The operator's section includes the 5G RAN DU operators for the 4.11.z release of OpenShift: LSO, PTP, SR-IOV, Logging and the Accelerator operator.
* The RHACM and MCE operators match the `--hub-version` version passed to the tool.


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
    - name: registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306
    - name: registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd
    - name: registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027
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

At this point, we can just modify the imageset definition to include new operators or additional images. On the other hand, we can just remove some of them if they are not going to be used. Another interesting idea is that we can replace operators or catalog sources to use the ones that are in a local or disconnected registry instead of the official Red Hat registry. That's what we are going to do:

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
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker --privileged --rm quay.io/openshift-kni/telco-ran-tools:latest -- factory-precaching-cli \
    download -r 4.11.5 --hub-version 2.5.4 -f /mnt \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306 \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027 \
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

```
# cp /tmp/eko4-ca.crt /etc/pki/ca-trust/source/anchors/.
# update-ca-trust 
```

Next, we just need to mount the host `/etc/pki` folder into the factory-precaching-cli container image. The factory-precaching-cli image is built on a UBI RHEL image, so paths and locations for certificates are going hand by hand with RHCOS (based on RHEL too). Take that into account when mounting host folders.

```console
# podman run -v /mnt:/mnt -v /root/.docker:/root/.docker -v /etc/pki:/etc/pki --privileged --rm quay.io/openshift-kni/telco-ran-tools:latest -- \
    factory-precaching-cli download -r 4.11.5 --hub-version 2.5.4 -f /mnt \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:da1753f9fcb9e229d0a68de03fac90d15023e647a8db531ae489eb93845d5306 \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-reporter-rhel8@sha256:e8d6b78248352b1a8e05a22308185a468d4a139682d997a7f968b329abbc02cd \
    --ai-img registry.redhat.io/multicluster-engine/assisted-installer-rhel8@sha256:33abd6e21cfdc36dd4337fa6f3c3442d33fc3f976471614dca5b8ef749e7a027 \
    --img quay.io/alosadag/troubleshoot \
    --du-profile -s \
    --skip-imageset
 
Generating list of pre-cached artifacts...
...
```
