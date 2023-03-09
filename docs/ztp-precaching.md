# Factory-cli: Zero Touch Provisioning configuration

<!-- vscode-markdown-toc -->

* 1. [Zero Touch Provisioning (ZTP)](#ZeroTouchProvisioningZTP)
* 2. [ZTP workflow](#ZTPworkflow)
* 3. [Pre-requisites](#Pre-requisites)
* 4. [Preparing the siteconfig](#Preparingthesiteconfig)
	* 4.1. [clusters.ignitionConfigOverride](#clusters.ignitionConfigOverride)
	* 4.2. [nodes.installerArgs](#nodes.installerArgs)
	* 4.3. [nodes.ignitionConfigOverride](#nodes.ignitionConfigOverride)
* 5. [Start ZTP](#StartZTP)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

>:exclamation: The factory-cli tool permits to download a rootfs image to the local partition. Therefore, when booting the discovery ISO we can use a lightweight image and pull the rootfs from a local partition or from a local HTTPd server. Currently ZTP does not allow a declarative and automated way to load the rootfs image from a local partition.

##  1. <a name='ZeroTouchProvisioningZTP'></a>Zero Touch Provisioning (ZTP)

Edge computing presents extraordinary challenges with managing hundreds to tens of thousands of clusters in hundreds of thousands of locations. These challenges require fully-automated management solutions with, as closely as possible, zero human interaction.

*Zero touch provisioning (ZTP)* allows you to provision new edge sites with declarative configurations of bare-metal equipment at remote sites following a GitOps deployment set of practices. All configurations are declarative in nature.

ZTP is a project to deploy and deliver OpenShift 4 in a hub-and-spoke architecture (in a relation of 1-N), where a single hub cluster manages many spoke clusters. The hub and the spokes will be based on OpenShift 4 but with the difference that the hub cluster will manage, deploy and control the lifecycle of the spokes using Red Hat Advanced Cluster Management (RHACM). So, hub clusters running RHACM apply radio access network (RAN) policies from predefined custom resources (CRs) and provision and deploy the spoke clusters using multiple products. 

>:exclamation: ZTP can have two scenarios, connected and disconnected, whether the OpenShift Container Platform Worker nodes can directly access the internet or not. In telco deployments, the disconnected scenario is the most common.

ZTP provides support for deploying single node clusters, three-node clusters, and standard OpenShift clusters. This includes the installation of OpenShift and deployment of the distributed units (DUs) at scale. However, the factory-cli tool focuses on SNO clusters only.

![ZTP framework](images/ztp_edge_framework.png "ZTP framework")

##  2. <a name='ZTPworkflow'></a>ZTP workflow

Zero Touch Provisioning (ZTP) leverages multiple products or components to deploy OpenShift Container Platform clusters using a GitOps approach. While the workflow starts when the site is connected to the network and ends with the CNF workload deployed and running on the site nodes, it can be logically divided into two different stages: provisioning of the SNO and applying the desired configuration, which in our case is applying the validated RAN DU profile.

> :warning: The workflow does not need any intervention, so ZTP automatically will configure the SNO once it is provisioned. However, two stages are clearly differentiated.

The workflow is officially started by creating declarative configurations for the provisioning of your OpenShift clusters. This manifest is described in a custom resource called `siteConfig`. See that in a disconnected environment, there is a need for a container registry which has been configured to deliver the required OpenShift container images required for the installation. This task can be achieved by using [oc-mirror])https://docs.openshift.com/container-platform/latest/installing/disconnected_install/installing-mirroring-disconnected.html).

>:exclamation: The factory-cli-tool tries to limit the usage of a container registry to pull down the required images. Currently, a registry is still needed since a couple of components require checking the availability of certain images to continue with the installation. The end goal is avoiding this requirement.

Depending on your specific environment, you might need a couple of extra services such as DHCP, DNS, NTP or HTTP. The latest will be needed for downloading the RHCOS live ISO and the rootfs image locally instead of the default [OpenShift mirror webpage](http://mirror.openshift.com.)

Once the configuration is created you can push it to the Git repo where Argo CD is continuously looking to pull the new content:

![ZTP workflow 0](images/ztp_workflow_0.png "ZTP workflow 0")

Argo CD pulls the siteConfig and uses a specific kustomize plugin called [siteconfig-generator](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/siteconfig-generator-kustomize-plugin) to transform it into custom resources that are understood by the hub cluster (RHACM/MCE). A siteConfig contains all the necessary information to provision your node or nodes. Basically, it will create ISO images with the defined configuration that are delivered to the edge nodes to begin the installation process. The images are used to repeatedly provision large numbers of nodes efficiently and quickly, allowing you to keep up with requirements from the field for far-edge nodes. 

> :warning: On telco use cases, clusters are mainly running on bare-metal hosts. Therefore the produced ISO images are mounted using remote virtual media features of the baseboard management controller (BMC).

In the picture, these resulting manifests are called Cluster Installation CRs. Finally, the provisioning process starts.

![ZTP workflow 1](images/ztp_workflow_1.png "ZTP workflow 1")

The provisioning process includes installing the host operating system (RHCOS) on a blank server and deploying OpenShift Container Platform. This stage is managed mainly by a ZTP component called the Infrastructure Operator

> :exclamation: Notice, in the picture, how ZTP allows us to provision clusters at scale.

![ZTP workflow 2](images/ztp_workflow_2.png "ZTP workflow 2")

Once the clusters are provisioned, the day-2 configuration defined in multiple `PolicyGenTemplate` (PGTs) custom resources will be automatically applied. `PolicyGenTemplate` custom resource is understood by the ZTP using a specific kustomize plugin called [policy-generator](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/policygenerator-kustomize-plugin). In telco RAN DU nodes, this configuration includes the installation of the common telco operators, a common configuration for RAN and specific configuration (SR-IOV or performance settings) for each site since it is very dependant on the hardware.

![ZTP workflow 3](images/ztp_workflow_3.png "ZTP workflow 3")

Notice that if, later on, you want to apply a new configuration or replace an existing configuration you must use a new `policyGenTemplate` to do that.


##  3. <a name='Pre-requisites'></a>Pre-requisites

* The [partitioning stage](./partitioning.md) was already executed successfully.
* The [downloading stage](./downloading.md) is completed, so all dependent artifacts are already stored on the disk partition.
* The bare metal server is powered off.

##  4. <a name='Preparingthesiteconfig'></a>Preparing the siteconfig

As mentioned, a `siteConfig` manifest defines in a declarative manner how an OpenShift target cluster is going to be installed and configured. Below it is an example of a valid siteConfig, however, unlike the regular ZTP provisioning workflow 3 extra fields need to be included:

* **clusters.ignitionConfigOverride**. This field adds an extra configuration in ignition format during the ZTP discovery stage. Basically, it includes a couple of systemd services in the ISO that it is mounted using virtual media. This way, those scripts are part of the RHCOS discovery live ISO and can be used at that point to load the Assisted Installer images.
* **nodes.installerArgs**. This field allows us to configure the way coreos-installer utility writes the RHCOS live ISO to disk. In this case, we need to indicate to save the disk partition labeled as 'data'. The artifacts saved there will be needed during the OCP installation stage.
* **nodes.ignitionConfigOverride**. This field adds similar functionality as the clusters.ignitionConfigOverride, but in the OCP installation stage. Notice that once the RHCOS is written to disk, the extra configuration included in the ZTP discovery ISO is not there anymore. It was in memory since we were running a live OS during the discovery stage. This field allows the addtion of extra configuration in ignition format to the coreos-installer binary, which is in charge of writing the RHCOS live OS to disk.

> :exclamation: You can just copy and paste the three fields described above in your siteConfig. A detailed explanation of each one is included in the following sections.

```yaml
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "clus3a-5g-lab"
  namespace: "clus3a-5g-lab"
spec:
  baseDomain: "e2e.bos.redhat.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusterImageSetNameRef: "img4.9.10-x86-64-appsub"
  sshPublicKey: "ssh-rsa ..."
  clusters:
  - clusterName: "sno-worker-0"
    clusterImageSetNameRef: "eko4-img4.11.5-x86-64-appsub"
    clusterLabels:
      group-du-sno: ""
      common-411: true
      sites : "clus3a-5g-lab"
      vendor: "OpenShift"
    clusterNetwork:
      - cidr: 10.128.0.0/14
        hostPrefix: 23
    machineNetwork:
      - cidr: 10.19.32.192/26
    serviceNetwork:
      - 172.30.0.0/16
    networkType: "OVNKubernetes"
    additionalNTPSources:
      - clock.corp.redhat.com
    ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Load prestaged images in discovery stage\n\nBefore=agent.service\n\n[Service]\nType=oneshot\nUser=root\nWorkingDirectory=/var/mnt\nExecStart=bash /usr/local/bin/extract-ai.sh\n#TimeoutStopSec=30\nExecStopPost=systemctl disable precache-images.service\n\n[Install]\nWantedBy=multi-user.target default.target\nWantedBy=agent.service"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":755,"user":{"name":"root"},"contents":{"source":"data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIwojIFV0aWxpdHkgZm9yIGxvYWRpbmcgcHJlc3RhZ2VkIGltYWdlcyBkdXJpbmcgbm9kZSBpbnN0YWxsYXRpb24KIwoKUFJPRz0kKGJhc2VuYW1lICIkMCIpCgpEQVRBRElSPSIvdG1wL3ByZXN0YWdpbmciCkZTPSIvZGV2L2Rpc2svYnktcGFydGxhYmVsL2RhdGEiCgojIERldGVybWluZSB0aGUgaW1hZ2UgbGlzdCBmcm9tIHRoZSBzY3JpcHQgbmFtZSwgZXh0cmFjdC1haS5zaCBvciBleHRyYWN0LW9jcC5zaApJTUdfR1JPVVA9JChlY2hvICIke1BST0d9IiB8IHNlZCAtciAncy8uKmV4dHJhY3QtKC4qKVwuc2gvXDEvJykKSU1HX0xJU1RfRklMRT0iJHtEQVRBRElSfS8ke0lNR19HUk9VUH0taW1hZ2VzLnR4dCIKTUFQUElOR19GSUxFPSIke0RBVEFESVJ9L21hcHBpbmcudHh0IgoKIyBTZXQgdGhlIHBhcmFsbGVsaXphdGlvbiBqb2IgcG9vbCBzaXplIHRvIDgwJSBvZiB0aGUgY29yZXMKQ1BVUz0kKG5wcm9jIC0tYWxsKQpNQVhfQ1BVX01VTFQ9MC44CkpPQl9QT09MX1NJWkU9JChqcSAtbiAiJENQVVMqJE1BWF9DUFVfTVVMVCIgfCBjdXQgLWQgLiAtZjEpCgojIEdldCBpbml0aWFsIHN0YXJ0aW5nIHBvaW50IGZvciBpbmZvIGxvZyBhdCBlbmQgb2YgZXhlY3V0aW9uClNUQVJUPSR7U0VDT05EU30KCiMKIyBjbGVhbnVwOiBDbGVhbiB1cCByZXNvdXJjZXMgb24gZXhpdAojCmZ1bmN0aW9uIGNsZWFudXAgewogICAgY2QgLwogICAgaWYgbW91bnRwb2ludCAtcSAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICB1bW91bnQgIiR7REFUQURJUn0iCiAgICBmaQoKICAgIHJtIC1yZiAiJHtEQVRBRElSfSIKfQoKdHJhcCBjbGVhbnVwIEVYSVQKCiMKIyBtb3VudF9kYXRhOgojCmZ1bmN0aW9uIG1vdW50X2RhdGEgewogICAgaWYgISBta2RpciAtcCAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIGNyZWF0ZSAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBpZiBbICEgLWIgIiR7RlN9IiBdOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIE5vdCBhIGJsb2NrIGRldmljZTogJHtGU30iCiAgICAgICAgZXhpdCAxCiAgICBmaQoKICAgIGlmICEgbW91bnQgIiR7RlN9IiAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIG1vdW50ICR7RlN9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBmb3IgZiBpbiAiJHtJTUdfTElTVF9GSUxFfSIgIiR7TUFQUElOR19GSUxFfSI7IGRvCiAgICAgICAgaWYgWyAhIC1mICIke2Z9IiBdOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSBDb3VsZCBub3QgZmluZCAke2Z9IgogICAgICAgICAgICBleGl0IDEKICAgICAgICBmaQogICAgZG9uZQoKICAgIGlmICEgcHVzaGQgIiR7REFUQURJUn0iOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIEZhaWxlZCB0byBjaGRpciB0byAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKfQoKIwojIGNvcHlfaW1hZ2U6IEZ1bmN0aW9uIHRoYXQgaGFuZGxlcyBleHRyYWN0aW5nIGFuIGltYWdlIHRhcmJhbGwgYW5kIGNvcHlpbmcgaXQgaW50byBjb250YWluZXIgc3RvcmFnZS4KIyAgICAgICAgICAgICBMYXVuY2hlZCBpbiBiYWNrZ3JvdW5kIGZvciBwYXJhbGxlbGl6YXRpb24sIG9yIGlubGluZSBmb3IgcmV0cmllcwojCmZ1bmN0aW9uIGNvcHlfaW1hZ2UgewogICAgbG9jYWwgY3VycmVudF9jb3B5PSQxCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JDIKICAgIGxvY2FsIHVyaT0kMwogICAgbG9jYWwgdGFnPSQ0CiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCBuYW1lPQoKICAgIGVjaG8gIiR7UFJPR306IFtERUJVR10gRXh0cmFjdGluZyBpbWFnZSAke3VyaX0iCiAgICBuYW1lPSQoYmFzZW5hbWUgIiR7dXJpLzovX30iKQogICAgaWYgISB0YXIgLS11c2UtY29tcHJlc3MtcHJvZ3JhbT1waWd6IC14ZiAiJHtuYW1lfS50Z3oiOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0VSUk9SXSBDb3VsZCBub3QgZXh0cmFjdCB0aGUgaW1hZ2UgJHtuYW1lfS50Z3oiCiAgICAgICAgcmV0dXJuIDEKICAgIGZpCgogICAgaWYgW1sgIiR7SU1HX0dST1VQfSIgPSAiYWkiICYmIC1uICIke3RhZ30iICYmICIke3VyaX0iID1+ICJAc2hhIiBdXTsgdGhlbgogICAgICAgICMgRHVyaW5nIHRoZSBBSSBsb2FkaW5nIHN0YWdlLCBpZiB0aGUgaW1hZ2UgaGFzIGEgdGFnLCBsb2FkIHRoYXQgaW50byBjb250YWluZXIgc3RvcmFnZSBhcyB3ZWxsCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0lORk9dIENvcHlpbmcgJHt1cml9LCB3aXRoIHRhZyAke3RhZ30gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIG5vdGFnPSR7dXJpL0AqfQogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEgJiYgXAogICAgICAgICAgICBza29wZW8gY29weSAtLXJldHJ5LXRpbWVzIDEwICJkaXI6Ly8ke1BXRH0vJHtuYW1lfSIgImNvbnRhaW5lcnMtc3RvcmFnZToke25vdGFnfToke3RhZ30iIC1xCiAgICAgICAgcmM9JD8KICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbSU5GT10gQ29weWluZyAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEKICAgICAgICByYz0kPwogICAgZmkKCiAgICBlY2hvICIke1BST0d9OiBbSU5GT10gUmVtb3ZpbmcgZm9sZGVyIGZvciAke3VyaX0iCiAgICBybSAtcmYgIiR7bmFtZX0iCgogICAgcmV0dXJuICR7cmN9Cn0KCiMKIyBsb2FkX2ltYWdlczogTGF1bmNoIGpvYnMgdG8gcHJlc3RhZ2UgaW1hZ2VzIGZyb20gdGhlIGFwcHJvcHJpYXRlIGxpc3QgZmlsZQojCmZ1bmN0aW9uIGxvYWRfaW1hZ2VzIHsKICAgIGxvY2FsIC1BIHBpZHMgIyBIYXNoIHRoYXQgaW5jbHVkZSB0aGUgaW1hZ2VzIHB1bGxlZCBhbG9uZyB3aXRoIHRoZWlyIHBpZHMgdG8gYmUgbW9uaXRvcmVkIGJ5IHdhaXQgY29tbWFuZAogICAgbG9jYWwgLWEgaW1hZ2VzCiAgICBtYXBmaWxlIC10IGltYWdlcyA8IDwoIHNvcnQgLXUgIiR7SU1HX0xJU1RfRklMRX0iICkKCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjaW1hZ2VzW0BdfQogICAgbG9jYWwgY3VycmVudF9jb3B5PTAKICAgIGxvY2FsIGpvYl9jb3VudD0wCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIFJlYWR5IHRvIGV4dHJhY3QgJHt0b3RhbF9jb3BpZXN9IGltYWdlcyB1c2luZyAkSk9CX1BPT0xfU0laRSBzaW11bHRhbmVvdXMgcHJvY2Vzc2VzIgoKICAgIGZvciB1cmkgaW4gIiR7aW1hZ2VzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgIyBDaGVjayB0aGF0IHdlJ3ZlIGdvdCBmcmVlIHNwYWNlIGluIHRoZSBqb2IgcG9vbAogICAgICAgIHdoaWxlIFsgIiR7am9iX2NvdW50fSIgLWdlICIke0pPQl9QT09MX1NJWkV9IiBdOyBkbwogICAgICAgICAgICBzbGVlcCAwLjEKICAgICAgICAgICAgam9iX2NvdW50PSQoam9icyB8IHdjIC1sKQogICAgICAgIGRvbmUKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0RFQlVHXSBQcm9jZXNzaW5nIGltYWdlICR7dXJpfSIKICAgICAgICBpZiBwb2RtYW4gaW1hZ2UgZXhpc3RzICIke3VyaX0iOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtJTkZPXSBTa2lwcGluZyBleGlzdGluZyBpbWFnZSAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgICAgICBjb250aW51ZQogICAgICAgIGZpCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IiAmCgogICAgICAgIHBpZHNbJHt1cml9XT0kISAjIEtlZXBpbmcgdHJhY2sgb2YgdGhlIFBJRCBhbmQgY29udGFpbmVyIGltYWdlIGluIGNhc2UgdGhlIHB1bGwgZmFpbHMKICAgIGRvbmUKCiAgICBlY2hvICIke1BST0d9OiBbREVCVUddIFdhaXRpbmcgZm9yIGpvYiBjb21wbGV0aW9uIgogICAgZm9yIGltZyBpbiAiJHshcGlkc1tAXX0iOyBkbwogICAgICAgICMgV2FpdCBmb3IgZWFjaCBiYWNrZ3JvdW5kIHRhc2sgKFBJRCkuIElmIGFueSBlcnJvciwgdGhlbiBjb3B5IHRoZSBpbWFnZSBpbiB0aGUgZmFpbGVkIGFycmF5IHNvIGl0IGNhbiBiZSByZXRyaWVkIGxhdGVyCiAgICAgICAgaWYgISB3YWl0ICIke3BpZHNbJGltZ119IjsgdGhlbgogICAgICAgICAgICBlY2hvICIke1BST0d9OiBbRVJST1JdIFB1bGwgZmFpbGVkIGZvciBjb250YWluZXIgaW1hZ2U6ICR7aW1nfSAuIFJldHJ5aW5nIGxhdGVyLi4uICIKICAgICAgICAgICAgZmFpbGVkX2NvcGllcys9KCIke2ltZ30iKSAjIEZhaWxlZCwgdGhlbiBhZGQgdGhlIGltYWdlIHRvIGJlIHJldHJpZXZlZCBsYXRlcgogICAgICAgIGZpCiAgICBkb25lCn0KCiMKIyByZXRyeV9pbWFnZXM6IFJldHJ5IGxvYWRpbmcgYW55IGZhaWxlZCBpbWFnZXMgaW50byBjb250YWluZXIgc3RvcmFnZQojCmZ1bmN0aW9uIHJldHJ5X2ltYWdlcyB7CiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjZmFpbGVkX2NvcGllc1tAXX0KCiAgICBpZiBbICIke3RvdGFsX2NvcGllc30iIC1lcSAwIF07IHRoZW4KICAgICAgICByZXR1cm4gMAogICAgZmkKCiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCB0YWcKICAgIGxvY2FsIGN1cnJlbnRfY29weT0wCgogICAgZWNobyAiJHtQUk9HfTogW1JFVFJZSU5HXSIKICAgIGZvciBmYWlsZWRfY29weSBpbiAiJHtmYWlsZWRfY29waWVzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW1JFVFJZXSBSZXRyeWluZyBmYWlsZWQgaW1hZ2UgcHVsbDogJHtmYWlsZWRfY29weX0iCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IgogICAgICAgIHJjPSQ/CiAgICBkb25lCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIEltYWdlIGxvYWQgZG9uZSIKICAgIHJldHVybiAiJHtyY30iCn0KCmlmIFtbICIke0JBU0hfU09VUkNFWzBdfSIgPSAiJHswfSIgXV07IHRoZW4KICAgIGZhaWxlZF9jb3BpZXM9KCkgIyBBcnJheSB0aGF0IHdpbGwgaW5jbHVkZSBhbGwgdGhlIGltYWdlcyB0aGF0IGZhaWxlZCB0byBiZSBwdWxsZWQKCiAgICBtb3VudF9kYXRhCgogICAgbG9hZF9pbWFnZXMKCiAgICBpZiAhIHJldHJ5X2ltYWdlczsgdGhlbgogICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSAkeyNmYWlsZWRfY29waWVzW0BdfSBpbWFnZXMgd2VyZSBub3QgbG9hZGVkIHN1Y2Nlc3NmdWxseSwgYWZ0ZXIgJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiICNudW1iZXIgb2YgZmFpbGluZyBpbWFnZXMKICAgICAgICBleGl0IDEKICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbU1VDQ0VTU10gQWxsIGltYWdlcyB3ZXJlIGxvYWRlZCwgaW4gJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiCiAgICAgICAgZXhpdCAwCiAgICBmaQpmaQo="}},{"overwrite":true,"path":"/usr/local/bin/agent-fix-bz1964591","mode":755,"user":{"name":"root"},"contents":{"source":"data:,%23%21%2Fusr%2Fbin%2Fsh%0A%0A%23%20This%20script%20is%20a%20workaround%20for%20bugzilla%201964591%20where%20symlinks%20inside%20%2Fvar%2Flib%2Fcontainers%2F%20get%0A%23%20corrupted%20under%20some%20circumstances.%0A%23%0A%23%20In%20order%20to%20let%20agent.service%20start%20correctly%20we%20are%20checking%20here%20whether%20the%20requested%0A%23%20container%20image%20exists%20and%20in%20case%20%22podman%20images%22%20returns%20an%20error%20we%20try%20removing%20the%20faulty%0A%23%20image.%0A%23%0A%23%20In%20such%20a%20scenario%20agent.service%20will%20detect%20the%20image%20is%20not%20present%20and%20pull%20it%20again.%20In%20case%0A%23%20the%20image%20is%20present%20and%20can%20be%20detected%20correctly%2C%20no%20any%20action%20is%20required.%0A%0AIMAGE%3D%24%28echo%20%241%20%7C%20sed%20%27s%2F%5B%40%3A%5D.%2A%2F%2F%27%29%0Apodman%20images%20%7C%20grep%20%24IMAGE%20%7C%7C%20podman%20rmi%20--force%20%241%20%7C%7C%20true"}}]}}'
    nodes:
      - hostName: "snonode.sno-worker-0.e2e.bos.redhat.com"
        role: "master"
        bmcAddress: "idrac-virtualmedia+https://10.19.28.53/redfish/v1/Systems/System.Embedded.1"
        bmcCredentialsName:
          name: "worker0-bmh-secret"
        bootMACAddress: "e4:43:4b:bd:90:46"
        bootMode: "UEFI"
        rootDeviceHints:
          deviceName: /dev/nvme0n1
        cpuset: "0-1,40-41"
        installerArgs: '["--save-partlabel", "data"]'
        ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Load prestaged OCP images into containers storage\nBefore=machine-config-daemon-pull.service nodeip-configuration.service\n\n[Service]\nType=oneshot\nUser=root\nWorkingDirectory=/var/mnt\nExecStart=bash /usr/local/bin/extract-ocp.sh\nTimeoutStopSec=60\nExecStopPost=systemctl disable precache-ocp-images.service\n\n[Install]\nWantedBy=multi-user.target\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":755,"user":{"name":"root"},"contents":{"source":"data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIwojIFV0aWxpdHkgZm9yIGxvYWRpbmcgcHJlc3RhZ2VkIGltYWdlcyBkdXJpbmcgbm9kZSBpbnN0YWxsYXRpb24KIwoKUFJPRz0kKGJhc2VuYW1lICIkMCIpCgpEQVRBRElSPSIvdG1wL3ByZXN0YWdpbmciCkZTPSIvZGV2L2Rpc2svYnktcGFydGxhYmVsL2RhdGEiCgojIERldGVybWluZSB0aGUgaW1hZ2UgbGlzdCBmcm9tIHRoZSBzY3JpcHQgbmFtZSwgZXh0cmFjdC1haS5zaCBvciBleHRyYWN0LW9jcC5zaApJTUdfR1JPVVA9JChlY2hvICIke1BST0d9IiB8IHNlZCAtciAncy8uKmV4dHJhY3QtKC4qKVwuc2gvXDEvJykKSU1HX0xJU1RfRklMRT0iJHtEQVRBRElSfS8ke0lNR19HUk9VUH0taW1hZ2VzLnR4dCIKTUFQUElOR19GSUxFPSIke0RBVEFESVJ9L21hcHBpbmcudHh0IgoKIyBTZXQgdGhlIHBhcmFsbGVsaXphdGlvbiBqb2IgcG9vbCBzaXplIHRvIDgwJSBvZiB0aGUgY29yZXMKQ1BVUz0kKG5wcm9jIC0tYWxsKQpNQVhfQ1BVX01VTFQ9MC44CkpPQl9QT09MX1NJWkU9JChqcSAtbiAiJENQVVMqJE1BWF9DUFVfTVVMVCIgfCBjdXQgLWQgLiAtZjEpCgojIEdldCBpbml0aWFsIHN0YXJ0aW5nIHBvaW50IGZvciBpbmZvIGxvZyBhdCBlbmQgb2YgZXhlY3V0aW9uClNUQVJUPSR7U0VDT05EU30KCiMKIyBjbGVhbnVwOiBDbGVhbiB1cCByZXNvdXJjZXMgb24gZXhpdAojCmZ1bmN0aW9uIGNsZWFudXAgewogICAgY2QgLwogICAgaWYgbW91bnRwb2ludCAtcSAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICB1bW91bnQgIiR7REFUQURJUn0iCiAgICBmaQoKICAgIHJtIC1yZiAiJHtEQVRBRElSfSIKfQoKdHJhcCBjbGVhbnVwIEVYSVQKCiMKIyBtb3VudF9kYXRhOgojCmZ1bmN0aW9uIG1vdW50X2RhdGEgewogICAgaWYgISBta2RpciAtcCAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIGNyZWF0ZSAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBpZiBbICEgLWIgIiR7RlN9IiBdOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIE5vdCBhIGJsb2NrIGRldmljZTogJHtGU30iCiAgICAgICAgZXhpdCAxCiAgICBmaQoKICAgIGlmICEgbW91bnQgIiR7RlN9IiAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIG1vdW50ICR7RlN9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBmb3IgZiBpbiAiJHtJTUdfTElTVF9GSUxFfSIgIiR7TUFQUElOR19GSUxFfSI7IGRvCiAgICAgICAgaWYgWyAhIC1mICIke2Z9IiBdOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSBDb3VsZCBub3QgZmluZCAke2Z9IgogICAgICAgICAgICBleGl0IDEKICAgICAgICBmaQogICAgZG9uZQoKICAgIGlmICEgcHVzaGQgIiR7REFUQURJUn0iOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIEZhaWxlZCB0byBjaGRpciB0byAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKfQoKIwojIGNvcHlfaW1hZ2U6IEZ1bmN0aW9uIHRoYXQgaGFuZGxlcyBleHRyYWN0aW5nIGFuIGltYWdlIHRhcmJhbGwgYW5kIGNvcHlpbmcgaXQgaW50byBjb250YWluZXIgc3RvcmFnZS4KIyAgICAgICAgICAgICBMYXVuY2hlZCBpbiBiYWNrZ3JvdW5kIGZvciBwYXJhbGxlbGl6YXRpb24sIG9yIGlubGluZSBmb3IgcmV0cmllcwojCmZ1bmN0aW9uIGNvcHlfaW1hZ2UgewogICAgbG9jYWwgY3VycmVudF9jb3B5PSQxCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JDIKICAgIGxvY2FsIHVyaT0kMwogICAgbG9jYWwgdGFnPSQ0CiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCBuYW1lPQoKICAgIGVjaG8gIiR7UFJPR306IFtERUJVR10gRXh0cmFjdGluZyBpbWFnZSAke3VyaX0iCiAgICBuYW1lPSQoYmFzZW5hbWUgIiR7dXJpLzovX30iKQogICAgaWYgISB0YXIgLS11c2UtY29tcHJlc3MtcHJvZ3JhbT1waWd6IC14ZiAiJHtuYW1lfS50Z3oiOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0VSUk9SXSBDb3VsZCBub3QgZXh0cmFjdCB0aGUgaW1hZ2UgJHtuYW1lfS50Z3oiCiAgICAgICAgcmV0dXJuIDEKICAgIGZpCgogICAgaWYgW1sgIiR7SU1HX0dST1VQfSIgPSAiYWkiICYmIC1uICIke3RhZ30iICYmICIke3VyaX0iID1+ICJAc2hhIiBdXTsgdGhlbgogICAgICAgICMgRHVyaW5nIHRoZSBBSSBsb2FkaW5nIHN0YWdlLCBpZiB0aGUgaW1hZ2UgaGFzIGEgdGFnLCBsb2FkIHRoYXQgaW50byBjb250YWluZXIgc3RvcmFnZSBhcyB3ZWxsCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0lORk9dIENvcHlpbmcgJHt1cml9LCB3aXRoIHRhZyAke3RhZ30gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIG5vdGFnPSR7dXJpL0AqfQogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEgJiYgXAogICAgICAgICAgICBza29wZW8gY29weSAtLXJldHJ5LXRpbWVzIDEwICJkaXI6Ly8ke1BXRH0vJHtuYW1lfSIgImNvbnRhaW5lcnMtc3RvcmFnZToke25vdGFnfToke3RhZ30iIC1xCiAgICAgICAgcmM9JD8KICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbSU5GT10gQ29weWluZyAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEKICAgICAgICByYz0kPwogICAgZmkKCiAgICBlY2hvICIke1BST0d9OiBbSU5GT10gUmVtb3ZpbmcgZm9sZGVyIGZvciAke3VyaX0iCiAgICBybSAtcmYgIiR7bmFtZX0iCgogICAgcmV0dXJuICR7cmN9Cn0KCiMKIyBsb2FkX2ltYWdlczogTGF1bmNoIGpvYnMgdG8gcHJlc3RhZ2UgaW1hZ2VzIGZyb20gdGhlIGFwcHJvcHJpYXRlIGxpc3QgZmlsZQojCmZ1bmN0aW9uIGxvYWRfaW1hZ2VzIHsKICAgIGxvY2FsIC1BIHBpZHMgIyBIYXNoIHRoYXQgaW5jbHVkZSB0aGUgaW1hZ2VzIHB1bGxlZCBhbG9uZyB3aXRoIHRoZWlyIHBpZHMgdG8gYmUgbW9uaXRvcmVkIGJ5IHdhaXQgY29tbWFuZAogICAgbG9jYWwgLWEgaW1hZ2VzCiAgICBtYXBmaWxlIC10IGltYWdlcyA8IDwoIHNvcnQgLXUgIiR7SU1HX0xJU1RfRklMRX0iICkKCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjaW1hZ2VzW0BdfQogICAgbG9jYWwgY3VycmVudF9jb3B5PTAKICAgIGxvY2FsIGpvYl9jb3VudD0wCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIFJlYWR5IHRvIGV4dHJhY3QgJHt0b3RhbF9jb3BpZXN9IGltYWdlcyB1c2luZyAkSk9CX1BPT0xfU0laRSBzaW11bHRhbmVvdXMgcHJvY2Vzc2VzIgoKICAgIGZvciB1cmkgaW4gIiR7aW1hZ2VzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgIyBDaGVjayB0aGF0IHdlJ3ZlIGdvdCBmcmVlIHNwYWNlIGluIHRoZSBqb2IgcG9vbAogICAgICAgIHdoaWxlIFsgIiR7am9iX2NvdW50fSIgLWdlICIke0pPQl9QT09MX1NJWkV9IiBdOyBkbwogICAgICAgICAgICBzbGVlcCAwLjEKICAgICAgICAgICAgam9iX2NvdW50PSQoam9icyB8IHdjIC1sKQogICAgICAgIGRvbmUKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0RFQlVHXSBQcm9jZXNzaW5nIGltYWdlICR7dXJpfSIKICAgICAgICBpZiBwb2RtYW4gaW1hZ2UgZXhpc3RzICIke3VyaX0iOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtJTkZPXSBTa2lwcGluZyBleGlzdGluZyBpbWFnZSAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgICAgICBjb250aW51ZQogICAgICAgIGZpCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IiAmCgogICAgICAgIHBpZHNbJHt1cml9XT0kISAjIEtlZXBpbmcgdHJhY2sgb2YgdGhlIFBJRCBhbmQgY29udGFpbmVyIGltYWdlIGluIGNhc2UgdGhlIHB1bGwgZmFpbHMKICAgIGRvbmUKCiAgICBlY2hvICIke1BST0d9OiBbREVCVUddIFdhaXRpbmcgZm9yIGpvYiBjb21wbGV0aW9uIgogICAgZm9yIGltZyBpbiAiJHshcGlkc1tAXX0iOyBkbwogICAgICAgICMgV2FpdCBmb3IgZWFjaCBiYWNrZ3JvdW5kIHRhc2sgKFBJRCkuIElmIGFueSBlcnJvciwgdGhlbiBjb3B5IHRoZSBpbWFnZSBpbiB0aGUgZmFpbGVkIGFycmF5IHNvIGl0IGNhbiBiZSByZXRyaWVkIGxhdGVyCiAgICAgICAgaWYgISB3YWl0ICIke3BpZHNbJGltZ119IjsgdGhlbgogICAgICAgICAgICBlY2hvICIke1BST0d9OiBbRVJST1JdIFB1bGwgZmFpbGVkIGZvciBjb250YWluZXIgaW1hZ2U6ICR7aW1nfSAuIFJldHJ5aW5nIGxhdGVyLi4uICIKICAgICAgICAgICAgZmFpbGVkX2NvcGllcys9KCIke2ltZ30iKSAjIEZhaWxlZCwgdGhlbiBhZGQgdGhlIGltYWdlIHRvIGJlIHJldHJpZXZlZCBsYXRlcgogICAgICAgIGZpCiAgICBkb25lCn0KCiMKIyByZXRyeV9pbWFnZXM6IFJldHJ5IGxvYWRpbmcgYW55IGZhaWxlZCBpbWFnZXMgaW50byBjb250YWluZXIgc3RvcmFnZQojCmZ1bmN0aW9uIHJldHJ5X2ltYWdlcyB7CiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjZmFpbGVkX2NvcGllc1tAXX0KCiAgICBpZiBbICIke3RvdGFsX2NvcGllc30iIC1lcSAwIF07IHRoZW4KICAgICAgICByZXR1cm4gMAogICAgZmkKCiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCB0YWcKICAgIGxvY2FsIGN1cnJlbnRfY29weT0wCgogICAgZWNobyAiJHtQUk9HfTogW1JFVFJZSU5HXSIKICAgIGZvciBmYWlsZWRfY29weSBpbiAiJHtmYWlsZWRfY29waWVzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW1JFVFJZXSBSZXRyeWluZyBmYWlsZWQgaW1hZ2UgcHVsbDogJHtmYWlsZWRfY29weX0iCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IgogICAgICAgIHJjPSQ/CiAgICBkb25lCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIEltYWdlIGxvYWQgZG9uZSIKICAgIHJldHVybiAiJHtyY30iCn0KCmlmIFtbICIke0JBU0hfU09VUkNFWzBdfSIgPSAiJHswfSIgXV07IHRoZW4KICAgIGZhaWxlZF9jb3BpZXM9KCkgIyBBcnJheSB0aGF0IHdpbGwgaW5jbHVkZSBhbGwgdGhlIGltYWdlcyB0aGF0IGZhaWxlZCB0byBiZSBwdWxsZWQKCiAgICBtb3VudF9kYXRhCgogICAgbG9hZF9pbWFnZXMKCiAgICBpZiAhIHJldHJ5X2ltYWdlczsgdGhlbgogICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSAkeyNmYWlsZWRfY29waWVzW0BdfSBpbWFnZXMgd2VyZSBub3QgbG9hZGVkIHN1Y2Nlc3NmdWxseSwgYWZ0ZXIgJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiICNudW1iZXIgb2YgZmFpbGluZyBpbWFnZXMKICAgICAgICBleGl0IDEKICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbU1VDQ0VTU10gQWxsIGltYWdlcyB3ZXJlIGxvYWRlZCwgaW4gJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiCiAgICAgICAgZXhpdCAwCiAgICBmaQpmaQo="}}]}}'
        nodeNetwork:
          config:
            interfaces:
              - name: ens1f0
                type: ethernet
                state: up
                macAddress: "e4:43:4b:bd:90:46"
                ipv4:
                  enabled: true
                  dhcp: true
                ipv6:
                  enabled: false
          interfaces:
            - name: "ens1f0"
              macAddress: "e4:43:4b:bd:90:46"
```

###  4.1. <a name='clusters.ignitionConfigOverride'></a>clusters.ignitionConfigOverride

Showing the content of the field in a prettier and cleaner way will help us to understand much better what is actually doing the ignition configuration:

* There are two systemd units (var-mnt.mount nad precache-images.services). The precache-images.service depends on the disk partition to be mounted in /var/mnt by the var-mnt unit. The precache-images service basically calls a script called `extract-ai.sh`. Notice that the precache-images must be executed before the `agent.service`, so it means that extracting the Assisted Installer (ai) images is done before the discovery stage starts.
* The `extract-ai.sh` script basically uncompresses and loads the images required in this stage from the disk partition to the local container storage. Once done, images can be used locally instead of pulled down from a registry. The decoded script can be found [here](./resources/extract-ai.sh)
* The `agent-fix-bz1964591` is currently a workaround for an issue with Assisted Installer. Essentially, Assisted Installer removes some container images if they are local, forcing them to be downloaded from the registry. In our scenario, we want to avoid that since we just loaded them into the local container storage. The decoded script can be found [here](./resources/agent-fix-bz1964591)

> :exclamation Sometimes you could need to modify the mentioned scripts or include a new ones. In such cases you can do so by adding them into the [discovery-beauty ignition template](./resources/discovery-beauty.ign). Finally, include the modified ignition file into the siteConfig manifest in the expected format.

 
``` { .json title=discovery-beauty.ign }
{
  "ignition": {
    "version": "3.1.0"
  },
  "systemd": {
    "units": [
      {
        "name": "precache-images.service",
        "enabled": true,
        "contents": "[Unit]\nDescription=Load prestaged images in discovery stage\n\nBefore=agent.service\n\n[Service]\nType=oneshot\nUser=root\nWorkingDirectory=/var/mnt\nExecStart=bash /usr/local/bin/extract-ai.sh\n#TimeoutStopSec=30\nExecStopPost=systemctl disable precache-images.service\n\n[Install]\nWantedBy=multi-user.target default.target\nWantedBy=agent.service"
      }
    ]
  },
  "storage": {
    "files": [
      {
        "overwrite": true,
        "path": "/usr/local/bin/extract-ai.sh",
        "mode": 755,
        "user": {
          "name": "root"
        },
        "contents": {
          "source": "data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIwojIFV0aWxpdHkgZm9yIGxvYWRpbmcgcHJlc3RhZ2VkIGltYWdlcyBkdXJpbmcgbm9kZSBpbnN0YWxsYXRpb24KIwoKUFJPRz0kKGJhc2VuYW1lICIkMCIpCgpEQVRBRElSPSIvdG1wL3ByZXN0YWdpbmciCkZTPSIvZGV2L2Rpc2svYnktcGFydGxhYmVsL2RhdGEiCgojIERldGVybWluZSB0aGUgaW1hZ2UgbGlzdCBmcm9tIHRoZSBzY3JpcHQgbmFtZSwgZXh0cmFjdC1haS5zaCBvciBleHRyYWN0LW9jcC5zaApJTUdfR1JPVVA9JChlY2hvICIke1BST0d9IiB8IHNlZCAtciAncy8uKmV4dHJhY3QtKC4qKVwuc2gvXDEvJykKSU1HX0xJU1RfRklMRT0iJHtEQVRBRElSfS8ke0lNR19HUk9VUH0taW1hZ2VzLnR4dCIKTUFQUElOR19GSUxFPSIke0RBVEFESVJ9L21hcHBpbmcudHh0IgoKIyBTZXQgdGhlIHBhcmFsbGVsaXphdGlvbiBqb2IgcG9vbCBzaXplIHRvIDgwJSBvZiB0aGUgY29yZXMKQ1BVUz0kKG5wcm9jIC0tYWxsKQpNQVhfQ1BVX01VTFQ9MC44CkpPQl9QT09MX1NJWkU9JChqcSAtbiAiJENQVVMqJE1BWF9DUFVfTVVMVCIgfCBjdXQgLWQgLiAtZjEpCgojIEdldCBpbml0aWFsIHN0YXJ0aW5nIHBvaW50IGZvciBpbmZvIGxvZyBhdCBlbmQgb2YgZXhlY3V0aW9uClNUQVJUPSR7U0VDT05EU30KCiMKIyBjbGVhbnVwOiBDbGVhbiB1cCByZXNvdXJjZXMgb24gZXhpdAojCmZ1bmN0aW9uIGNsZWFudXAgewogICAgY2QgLwogICAgaWYgbW91bnRwb2ludCAtcSAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICB1bW91bnQgIiR7REFUQURJUn0iCiAgICBmaQoKICAgIHJtIC1yZiAiJHtEQVRBRElSfSIKfQoKdHJhcCBjbGVhbnVwIEVYSVQKCiMKIyBtb3VudF9kYXRhOgojCmZ1bmN0aW9uIG1vdW50X2RhdGEgewogICAgaWYgISBta2RpciAtcCAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIGNyZWF0ZSAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBpZiBbICEgLWIgIiR7RlN9IiBdOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIE5vdCBhIGJsb2NrIGRldmljZTogJHtGU30iCiAgICAgICAgZXhpdCAxCiAgICBmaQoKICAgIGlmICEgbW91bnQgIiR7RlN9IiAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIG1vdW50ICR7RlN9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBmb3IgZiBpbiAiJHtJTUdfTElTVF9GSUxFfSIgIiR7TUFQUElOR19GSUxFfSI7IGRvCiAgICAgICAgaWYgWyAhIC1mICIke2Z9IiBdOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSBDb3VsZCBub3QgZmluZCAke2Z9IgogICAgICAgICAgICBleGl0IDEKICAgICAgICBmaQogICAgZG9uZQoKICAgIGlmICEgcHVzaGQgIiR7REFUQURJUn0iOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIEZhaWxlZCB0byBjaGRpciB0byAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKfQoKIwojIGNvcHlfaW1hZ2U6IEZ1bmN0aW9uIHRoYXQgaGFuZGxlcyBleHRyYWN0aW5nIGFuIGltYWdlIHRhcmJhbGwgYW5kIGNvcHlpbmcgaXQgaW50byBjb250YWluZXIgc3RvcmFnZS4KIyAgICAgICAgICAgICBMYXVuY2hlZCBpbiBiYWNrZ3JvdW5kIGZvciBwYXJhbGxlbGl6YXRpb24sIG9yIGlubGluZSBmb3IgcmV0cmllcwojCmZ1bmN0aW9uIGNvcHlfaW1hZ2UgewogICAgbG9jYWwgY3VycmVudF9jb3B5PSQxCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JDIKICAgIGxvY2FsIHVyaT0kMwogICAgbG9jYWwgdGFnPSQ0CiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCBuYW1lPQoKICAgIGVjaG8gIiR7UFJPR306IFtERUJVR10gRXh0cmFjdGluZyBpbWFnZSAke3VyaX0iCiAgICBuYW1lPSQoYmFzZW5hbWUgIiR7dXJpLzovX30iKQogICAgaWYgISB0YXIgLS11c2UtY29tcHJlc3MtcHJvZ3JhbT1waWd6IC14ZiAiJHtuYW1lfS50Z3oiOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0VSUk9SXSBDb3VsZCBub3QgZXh0cmFjdCB0aGUgaW1hZ2UgJHtuYW1lfS50Z3oiCiAgICAgICAgcmV0dXJuIDEKICAgIGZpCgogICAgaWYgW1sgIiR7SU1HX0dST1VQfSIgPSAiYWkiICYmIC1uICIke3RhZ30iICYmICIke3VyaX0iID1+ICJAc2hhIiBdXTsgdGhlbgogICAgICAgICMgRHVyaW5nIHRoZSBBSSBsb2FkaW5nIHN0YWdlLCBpZiB0aGUgaW1hZ2UgaGFzIGEgdGFnLCBsb2FkIHRoYXQgaW50byBjb250YWluZXIgc3RvcmFnZSBhcyB3ZWxsCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0lORk9dIENvcHlpbmcgJHt1cml9LCB3aXRoIHRhZyAke3RhZ30gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIG5vdGFnPSR7dXJpL0AqfQogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEgJiYgXAogICAgICAgICAgICBza29wZW8gY29weSAtLXJldHJ5LXRpbWVzIDEwICJkaXI6Ly8ke1BXRH0vJHtuYW1lfSIgImNvbnRhaW5lcnMtc3RvcmFnZToke25vdGFnfToke3RhZ30iIC1xCiAgICAgICAgcmM9JD8KICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbSU5GT10gQ29weWluZyAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEKICAgICAgICByYz0kPwogICAgZmkKCiAgICBlY2hvICIke1BST0d9OiBbSU5GT10gUmVtb3ZpbmcgZm9sZGVyIGZvciAke3VyaX0iCiAgICBybSAtcmYgIiR7bmFtZX0iCgogICAgcmV0dXJuICR7cmN9Cn0KCiMKIyBsb2FkX2ltYWdlczogTGF1bmNoIGpvYnMgdG8gcHJlc3RhZ2UgaW1hZ2VzIGZyb20gdGhlIGFwcHJvcHJpYXRlIGxpc3QgZmlsZQojCmZ1bmN0aW9uIGxvYWRfaW1hZ2VzIHsKICAgIGxvY2FsIC1BIHBpZHMgIyBIYXNoIHRoYXQgaW5jbHVkZSB0aGUgaW1hZ2VzIHB1bGxlZCBhbG9uZyB3aXRoIHRoZWlyIHBpZHMgdG8gYmUgbW9uaXRvcmVkIGJ5IHdhaXQgY29tbWFuZAogICAgbG9jYWwgLWEgaW1hZ2VzCiAgICBtYXBmaWxlIC10IGltYWdlcyA8IDwoIHNvcnQgLXUgIiR7SU1HX0xJU1RfRklMRX0iICkKCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjaW1hZ2VzW0BdfQogICAgbG9jYWwgY3VycmVudF9jb3B5PTAKICAgIGxvY2FsIGpvYl9jb3VudD0wCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIFJlYWR5IHRvIGV4dHJhY3QgJHt0b3RhbF9jb3BpZXN9IGltYWdlcyB1c2luZyAkSk9CX1BPT0xfU0laRSBzaW11bHRhbmVvdXMgcHJvY2Vzc2VzIgoKICAgIGZvciB1cmkgaW4gIiR7aW1hZ2VzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgIyBDaGVjayB0aGF0IHdlJ3ZlIGdvdCBmcmVlIHNwYWNlIGluIHRoZSBqb2IgcG9vbAogICAgICAgIHdoaWxlIFsgIiR7am9iX2NvdW50fSIgLWdlICIke0pPQl9QT09MX1NJWkV9IiBdOyBkbwogICAgICAgICAgICBzbGVlcCAwLjEKICAgICAgICAgICAgam9iX2NvdW50PSQoam9icyB8IHdjIC1sKQogICAgICAgIGRvbmUKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0RFQlVHXSBQcm9jZXNzaW5nIGltYWdlICR7dXJpfSIKICAgICAgICBpZiBwb2RtYW4gaW1hZ2UgZXhpc3RzICIke3VyaX0iOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtJTkZPXSBTa2lwcGluZyBleGlzdGluZyBpbWFnZSAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgICAgICBjb250aW51ZQogICAgICAgIGZpCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IiAmCgogICAgICAgIHBpZHNbJHt1cml9XT0kISAjIEtlZXBpbmcgdHJhY2sgb2YgdGhlIFBJRCBhbmQgY29udGFpbmVyIGltYWdlIGluIGNhc2UgdGhlIHB1bGwgZmFpbHMKICAgIGRvbmUKCiAgICBlY2hvICIke1BST0d9OiBbREVCVUddIFdhaXRpbmcgZm9yIGpvYiBjb21wbGV0aW9uIgogICAgZm9yIGltZyBpbiAiJHshcGlkc1tAXX0iOyBkbwogICAgICAgICMgV2FpdCBmb3IgZWFjaCBiYWNrZ3JvdW5kIHRhc2sgKFBJRCkuIElmIGFueSBlcnJvciwgdGhlbiBjb3B5IHRoZSBpbWFnZSBpbiB0aGUgZmFpbGVkIGFycmF5IHNvIGl0IGNhbiBiZSByZXRyaWVkIGxhdGVyCiAgICAgICAgaWYgISB3YWl0ICIke3BpZHNbJGltZ119IjsgdGhlbgogICAgICAgICAgICBlY2hvICIke1BST0d9OiBbRVJST1JdIFB1bGwgZmFpbGVkIGZvciBjb250YWluZXIgaW1hZ2U6ICR7aW1nfSAuIFJldHJ5aW5nIGxhdGVyLi4uICIKICAgICAgICAgICAgZmFpbGVkX2NvcGllcys9KCIke2ltZ30iKSAjIEZhaWxlZCwgdGhlbiBhZGQgdGhlIGltYWdlIHRvIGJlIHJldHJpZXZlZCBsYXRlcgogICAgICAgIGZpCiAgICBkb25lCn0KCiMKIyByZXRyeV9pbWFnZXM6IFJldHJ5IGxvYWRpbmcgYW55IGZhaWxlZCBpbWFnZXMgaW50byBjb250YWluZXIgc3RvcmFnZQojCmZ1bmN0aW9uIHJldHJ5X2ltYWdlcyB7CiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjZmFpbGVkX2NvcGllc1tAXX0KCiAgICBpZiBbICIke3RvdGFsX2NvcGllc30iIC1lcSAwIF07IHRoZW4KICAgICAgICByZXR1cm4gMAogICAgZmkKCiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCB0YWcKICAgIGxvY2FsIGN1cnJlbnRfY29weT0wCgogICAgZWNobyAiJHtQUk9HfTogW1JFVFJZSU5HXSIKICAgIGZvciBmYWlsZWRfY29weSBpbiAiJHtmYWlsZWRfY29waWVzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW1JFVFJZXSBSZXRyeWluZyBmYWlsZWQgaW1hZ2UgcHVsbDogJHtmYWlsZWRfY29weX0iCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IgogICAgICAgIHJjPSQ/CiAgICBkb25lCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIEltYWdlIGxvYWQgZG9uZSIKICAgIHJldHVybiAiJHtyY30iCn0KCmlmIFtbICIke0JBU0hfU09VUkNFWzBdfSIgPSAiJHswfSIgXV07IHRoZW4KICAgIGZhaWxlZF9jb3BpZXM9KCkgIyBBcnJheSB0aGF0IHdpbGwgaW5jbHVkZSBhbGwgdGhlIGltYWdlcyB0aGF0IGZhaWxlZCB0byBiZSBwdWxsZWQKCiAgICBtb3VudF9kYXRhCgogICAgbG9hZF9pbWFnZXMKCiAgICBpZiAhIHJldHJ5X2ltYWdlczsgdGhlbgogICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSAkeyNmYWlsZWRfY29waWVzW0BdfSBpbWFnZXMgd2VyZSBub3QgbG9hZGVkIHN1Y2Nlc3NmdWxseSwgYWZ0ZXIgJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiICNudW1iZXIgb2YgZmFpbGluZyBpbWFnZXMKICAgICAgICBleGl0IDEKICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbU1VDQ0VTU10gQWxsIGltYWdlcyB3ZXJlIGxvYWRlZCwgaW4gJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiCiAgICAgICAgZXhpdCAwCiAgICBmaQpmaQo="
        }
      },
      {
        "overwrite": true,
        "path": "/usr/local/bin/agent-fix-bz1964591",
        "mode": 755,
        "user": {
          "name": "root"
        },
        "contents": {
          "source": "data:,%23%21%2Fusr%2Fbin%2Fsh%0A%0A%23%20This%20script%20is%20a%20workaround%20for%20bugzilla%201964591%20where%20symlinks%20inside%20%2Fvar%2Flib%2Fcontainers%2F%20get%0A%23%20corrupted%20under%20some%20circumstances.%0A%23%0A%23%20In%20order%20to%20let%20agent.service%20start%20correctly%20we%20are%20checking%20here%20whether%20the%20requested%0A%23%20container%20image%20exists%20and%20in%20case%20%22podman%20images%22%20returns%20an%20error%20we%20try%20removing%20the%20faulty%0A%23%20image.%0A%23%0A%23%20In%20such%20a%20scenario%20agent.service%20will%20detect%20the%20image%20is%20not%20present%20and%20pull%20it%20again.%20In%20case%0A%23%20the%20image%20is%20present%20and%20can%20be%20detected%20correctly%2C%20no%20any%20action%20is%20required.%0A%0AIMAGE%3D%24%28echo%20%241%20%7C%20sed%20%27s%2F%5B%40%3A%5D.%2A%2F%2F%27%29%0Apodman%20images%20%7C%20grep%20%24IMAGE%20%7C%7C%20podman%20rmi%20--force%20%241%20%7C%7C%20true"
        }
      }
    ]
  }
}
```

###  4.2. <a name='nodes.installerArgs'></a>nodes.installerArgs

This field, as mentioned, permits us to save the disk partition where all the container images were stored. Those extra parameters are passed directly to the `coreos-installer` binary who is in charge of writing the live RHCOS to disk. Then, on the next boot, the OS is executed from the disk. Notice that we previously named the [partition](./partitioning.md#create-the-partition) as data.

```
installerArgs: '["--save-partlabel", "data"]'
```

Several extra options can be passed to the coreos-installer utility. Here below, you can see the most interesting ones:

```
# coreos-installer install --help
coreos-installer-install 0.12.0
Install Fedora CoreOS or RHEL CoreOS

USAGE:
    coreos-installer install [OPTIONS] <dest-device>

OPTIONS:
...
    -u, --image-url <URL>           
            Manually specify the image URL

    -f, --image-file <path>         
            Manually specify a local image file

    -i, --ignition-file <path>      
            Embed an Ignition config from a file

    -I, --ignition-url <URL>        
            Embed an Ignition config from a URL
...            
        --save-partlabel <lx>...    
            Save partitions with this label glob

        --save-partindex <id>...    
            Save partitions with this number or range
...
        --insecure-ignition         
            Allow Ignition URL without HTTPS or hash
...
ARGS:
    <dest-device>    
            Destination device
```

###  4.3. <a name='nodes.ignitionConfigOverride'></a>nodes.ignitionConfigOverride

The purpose of this field is very similar to [clusters.ignitionConfigOverride](./ztp-config.md#clustersignitionconfigoverride). Unlike the previous ignitionOverride, we are in the OCP installation stage. This means that we need to extract and load the OCP images that are needed to install the cluster. Remember that in the discovery stage we are only taking care of the container images required for discovery. 

> :warning: The number of container images extracted and loaded is way bigger than in the discovery stage. So, depending on the OCP release and whether it was requested to install the telco operators, the time it takes will vary. 

* There are two systemd units (var-mnt.mount and precache-ocp.services). The precache-ocp.service depends on the disk partition to be mounted in /var/mnt by the var-mnt unit. The precache-ocp service basically calls a script called `extract-ocp.sh`. Notice that the precache-ocp must be executed before the `machine-config-daemon-pull.service` and `nodeip-configuration.service`, so it means that extracting the images is done before the OCP installation starts.
* The `extract-ocp.sh` script basically uncompresses and loads the images required in this stage from the disk partition to the local container storage. Fundamentally there are OCP release images and operators if they were requested to be installed. Once done, images can be used locally instead of pulled down from a registry. The decoded script can be found [here](./resources/extract-ocp.sh)

> :exclamation Sometimes you could need to modify the mentioned scripts or include a new ones. In such cases you can do so by adding them into the [boot-beauty ignition template](./resources/boot-beauty.ign). Finally, include the modified ignition file into the siteConfig manifest in the expected format.


``` { .json title=boot-beauty.ign }
{
  "ignition": {
    "version": "3.1.0"
  },
  "systemd": {
    "units": [
      {
        "name": "precache-ocp-images.service",
        "enabled": true,
        "contents": "[Unit]\nDescription=Load prestaged OCP images into containers storage\nBefore=machine-config-daemon-pull.service nodeip-configuration.service\n\n[Service]\nType=oneshot\nUser=root\nWorkingDirectory=/var/mnt\nExecStart=bash /usr/local/bin/extract-ocp.sh\nTimeoutStopSec=60\nExecStopPost=systemctl disable precache-ocp-images.service\n\n[Install]\nWantedBy=multi-user.target\n"
      }
    ]
  },
  "storage": {
    "files": [
      {
        "overwrite": true,
        "path": "/usr/local/bin/extract-ocp.sh",
        "mode": 755,
        "user": {
          "name": "root"
        },
        "contents": {
          "source": "data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKIwojIFV0aWxpdHkgZm9yIGxvYWRpbmcgcHJlc3RhZ2VkIGltYWdlcyBkdXJpbmcgbm9kZSBpbnN0YWxsYXRpb24KIwoKUFJPRz0kKGJhc2VuYW1lICIkMCIpCgpEQVRBRElSPSIvdG1wL3ByZXN0YWdpbmciCkZTPSIvZGV2L2Rpc2svYnktcGFydGxhYmVsL2RhdGEiCgojIERldGVybWluZSB0aGUgaW1hZ2UgbGlzdCBmcm9tIHRoZSBzY3JpcHQgbmFtZSwgZXh0cmFjdC1haS5zaCBvciBleHRyYWN0LW9jcC5zaApJTUdfR1JPVVA9JChlY2hvICIke1BST0d9IiB8IHNlZCAtciAncy8uKmV4dHJhY3QtKC4qKVwuc2gvXDEvJykKSU1HX0xJU1RfRklMRT0iJHtEQVRBRElSfS8ke0lNR19HUk9VUH0taW1hZ2VzLnR4dCIKTUFQUElOR19GSUxFPSIke0RBVEFESVJ9L21hcHBpbmcudHh0IgoKIyBTZXQgdGhlIHBhcmFsbGVsaXphdGlvbiBqb2IgcG9vbCBzaXplIHRvIDgwJSBvZiB0aGUgY29yZXMKQ1BVUz0kKG5wcm9jIC0tYWxsKQpNQVhfQ1BVX01VTFQ9MC44CkpPQl9QT09MX1NJWkU9JChqcSAtbiAiJENQVVMqJE1BWF9DUFVfTVVMVCIgfCBjdXQgLWQgLiAtZjEpCgojIEdldCBpbml0aWFsIHN0YXJ0aW5nIHBvaW50IGZvciBpbmZvIGxvZyBhdCBlbmQgb2YgZXhlY3V0aW9uClNUQVJUPSR7U0VDT05EU30KCiMKIyBjbGVhbnVwOiBDbGVhbiB1cCByZXNvdXJjZXMgb24gZXhpdAojCmZ1bmN0aW9uIGNsZWFudXAgewogICAgY2QgLwogICAgaWYgbW91bnRwb2ludCAtcSAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICB1bW91bnQgIiR7REFUQURJUn0iCiAgICBmaQoKICAgIHJtIC1yZiAiJHtEQVRBRElSfSIKfQoKdHJhcCBjbGVhbnVwIEVYSVQKCiMKIyBtb3VudF9kYXRhOgojCmZ1bmN0aW9uIG1vdW50X2RhdGEgewogICAgaWYgISBta2RpciAtcCAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIGNyZWF0ZSAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBpZiBbICEgLWIgIiR7RlN9IiBdOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIE5vdCBhIGJsb2NrIGRldmljZTogJHtGU30iCiAgICAgICAgZXhpdCAxCiAgICBmaQoKICAgIGlmICEgbW91bnQgIiR7RlN9IiAiJHtEQVRBRElSfSI7IHRoZW4KICAgICAgICBlY2hvICIke1BST0d9OiBbRkFJTF0gRmFpbGVkIHRvIG1vdW50ICR7RlN9IgogICAgICAgIGV4aXQgMQogICAgZmkKCiAgICBmb3IgZiBpbiAiJHtJTUdfTElTVF9GSUxFfSIgIiR7TUFQUElOR19GSUxFfSI7IGRvCiAgICAgICAgaWYgWyAhIC1mICIke2Z9IiBdOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSBDb3VsZCBub3QgZmluZCAke2Z9IgogICAgICAgICAgICBleGl0IDEKICAgICAgICBmaQogICAgZG9uZQoKICAgIGlmICEgcHVzaGQgIiR7REFUQURJUn0iOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0ZBSUxdIEZhaWxlZCB0byBjaGRpciB0byAke0RBVEFESVJ9IgogICAgICAgIGV4aXQgMQogICAgZmkKfQoKIwojIGNvcHlfaW1hZ2U6IEZ1bmN0aW9uIHRoYXQgaGFuZGxlcyBleHRyYWN0aW5nIGFuIGltYWdlIHRhcmJhbGwgYW5kIGNvcHlpbmcgaXQgaW50byBjb250YWluZXIgc3RvcmFnZS4KIyAgICAgICAgICAgICBMYXVuY2hlZCBpbiBiYWNrZ3JvdW5kIGZvciBwYXJhbGxlbGl6YXRpb24sIG9yIGlubGluZSBmb3IgcmV0cmllcwojCmZ1bmN0aW9uIGNvcHlfaW1hZ2UgewogICAgbG9jYWwgY3VycmVudF9jb3B5PSQxCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JDIKICAgIGxvY2FsIHVyaT0kMwogICAgbG9jYWwgdGFnPSQ0CiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCBuYW1lPQoKICAgIGVjaG8gIiR7UFJPR306IFtERUJVR10gRXh0cmFjdGluZyBpbWFnZSAke3VyaX0iCiAgICBuYW1lPSQoYmFzZW5hbWUgIiR7dXJpLzovX30iKQogICAgaWYgISB0YXIgLS11c2UtY29tcHJlc3MtcHJvZ3JhbT1waWd6IC14ZiAiJHtuYW1lfS50Z3oiOyB0aGVuCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0VSUk9SXSBDb3VsZCBub3QgZXh0cmFjdCB0aGUgaW1hZ2UgJHtuYW1lfS50Z3oiCiAgICAgICAgcmV0dXJuIDEKICAgIGZpCgogICAgaWYgW1sgIiR7SU1HX0dST1VQfSIgPSAiYWkiICYmIC1uICIke3RhZ30iICYmICIke3VyaX0iID1+ICJAc2hhIiBdXTsgdGhlbgogICAgICAgICMgRHVyaW5nIHRoZSBBSSBsb2FkaW5nIHN0YWdlLCBpZiB0aGUgaW1hZ2UgaGFzIGEgdGFnLCBsb2FkIHRoYXQgaW50byBjb250YWluZXIgc3RvcmFnZSBhcyB3ZWxsCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0lORk9dIENvcHlpbmcgJHt1cml9LCB3aXRoIHRhZyAke3RhZ30gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIG5vdGFnPSR7dXJpL0AqfQogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEgJiYgXAogICAgICAgICAgICBza29wZW8gY29weSAtLXJldHJ5LXRpbWVzIDEwICJkaXI6Ly8ke1BXRH0vJHtuYW1lfSIgImNvbnRhaW5lcnMtc3RvcmFnZToke25vdGFnfToke3RhZ30iIC1xCiAgICAgICAgcmM9JD8KICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbSU5GT10gQ29weWluZyAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgIHNrb3BlbyBjb3B5IC0tcmV0cnktdGltZXMgMTAgImRpcjovLyR7UFdEfS8ke25hbWV9IiAiY29udGFpbmVycy1zdG9yYWdlOiR7dXJpfSIgLXEKICAgICAgICByYz0kPwogICAgZmkKCiAgICBlY2hvICIke1BST0d9OiBbSU5GT10gUmVtb3ZpbmcgZm9sZGVyIGZvciAke3VyaX0iCiAgICBybSAtcmYgIiR7bmFtZX0iCgogICAgcmV0dXJuICR7cmN9Cn0KCiMKIyBsb2FkX2ltYWdlczogTGF1bmNoIGpvYnMgdG8gcHJlc3RhZ2UgaW1hZ2VzIGZyb20gdGhlIGFwcHJvcHJpYXRlIGxpc3QgZmlsZQojCmZ1bmN0aW9uIGxvYWRfaW1hZ2VzIHsKICAgIGxvY2FsIC1BIHBpZHMgIyBIYXNoIHRoYXQgaW5jbHVkZSB0aGUgaW1hZ2VzIHB1bGxlZCBhbG9uZyB3aXRoIHRoZWlyIHBpZHMgdG8gYmUgbW9uaXRvcmVkIGJ5IHdhaXQgY29tbWFuZAogICAgbG9jYWwgLWEgaW1hZ2VzCiAgICBtYXBmaWxlIC10IGltYWdlcyA8IDwoIHNvcnQgLXUgIiR7SU1HX0xJU1RfRklMRX0iICkKCiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjaW1hZ2VzW0BdfQogICAgbG9jYWwgY3VycmVudF9jb3B5PTAKICAgIGxvY2FsIGpvYl9jb3VudD0wCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIFJlYWR5IHRvIGV4dHJhY3QgJHt0b3RhbF9jb3BpZXN9IGltYWdlcyB1c2luZyAkSk9CX1BPT0xfU0laRSBzaW11bHRhbmVvdXMgcHJvY2Vzc2VzIgoKICAgIGZvciB1cmkgaW4gIiR7aW1hZ2VzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgIyBDaGVjayB0aGF0IHdlJ3ZlIGdvdCBmcmVlIHNwYWNlIGluIHRoZSBqb2IgcG9vbAogICAgICAgIHdoaWxlIFsgIiR7am9iX2NvdW50fSIgLWdlICIke0pPQl9QT09MX1NJWkV9IiBdOyBkbwogICAgICAgICAgICBzbGVlcCAwLjEKICAgICAgICAgICAgam9iX2NvdW50PSQoam9icyB8IHdjIC1sKQogICAgICAgIGRvbmUKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW0RFQlVHXSBQcm9jZXNzaW5nIGltYWdlICR7dXJpfSIKICAgICAgICBpZiBwb2RtYW4gaW1hZ2UgZXhpc3RzICIke3VyaX0iOyB0aGVuCiAgICAgICAgICAgIGVjaG8gIiR7UFJPR306IFtJTkZPXSBTa2lwcGluZyBleGlzdGluZyBpbWFnZSAke3VyaX0gWyR7Y3VycmVudF9jb3B5fS8ke3RvdGFsX2NvcGllc31dIgogICAgICAgICAgICBjb250aW51ZQogICAgICAgIGZpCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IiAmCgogICAgICAgIHBpZHNbJHt1cml9XT0kISAjIEtlZXBpbmcgdHJhY2sgb2YgdGhlIFBJRCBhbmQgY29udGFpbmVyIGltYWdlIGluIGNhc2UgdGhlIHB1bGwgZmFpbHMKICAgIGRvbmUKCiAgICBlY2hvICIke1BST0d9OiBbREVCVUddIFdhaXRpbmcgZm9yIGpvYiBjb21wbGV0aW9uIgogICAgZm9yIGltZyBpbiAiJHshcGlkc1tAXX0iOyBkbwogICAgICAgICMgV2FpdCBmb3IgZWFjaCBiYWNrZ3JvdW5kIHRhc2sgKFBJRCkuIElmIGFueSBlcnJvciwgdGhlbiBjb3B5IHRoZSBpbWFnZSBpbiB0aGUgZmFpbGVkIGFycmF5IHNvIGl0IGNhbiBiZSByZXRyaWVkIGxhdGVyCiAgICAgICAgaWYgISB3YWl0ICIke3BpZHNbJGltZ119IjsgdGhlbgogICAgICAgICAgICBlY2hvICIke1BST0d9OiBbRVJST1JdIFB1bGwgZmFpbGVkIGZvciBjb250YWluZXIgaW1hZ2U6ICR7aW1nfSAuIFJldHJ5aW5nIGxhdGVyLi4uICIKICAgICAgICAgICAgZmFpbGVkX2NvcGllcys9KCIke2ltZ30iKSAjIEZhaWxlZCwgdGhlbiBhZGQgdGhlIGltYWdlIHRvIGJlIHJldHJpZXZlZCBsYXRlcgogICAgICAgIGZpCiAgICBkb25lCn0KCiMKIyByZXRyeV9pbWFnZXM6IFJldHJ5IGxvYWRpbmcgYW55IGZhaWxlZCBpbWFnZXMgaW50byBjb250YWluZXIgc3RvcmFnZQojCmZ1bmN0aW9uIHJldHJ5X2ltYWdlcyB7CiAgICBsb2NhbCB0b3RhbF9jb3BpZXM9JHsjZmFpbGVkX2NvcGllc1tAXX0KCiAgICBpZiBbICIke3RvdGFsX2NvcGllc30iIC1lcSAwIF07IHRoZW4KICAgICAgICByZXR1cm4gMAogICAgZmkKCiAgICBsb2NhbCByYz0wCiAgICBsb2NhbCB0YWcKICAgIGxvY2FsIGN1cnJlbnRfY29weT0wCgogICAgZWNobyAiJHtQUk9HfTogW1JFVFJZSU5HXSIKICAgIGZvciBmYWlsZWRfY29weSBpbiAiJHtmYWlsZWRfY29waWVzW0BdfSI7IGRvCiAgICAgICAgY3VycmVudF9jb3B5PSQoKGN1cnJlbnRfY29weSsxKSkKCiAgICAgICAgZWNobyAiJHtQUk9HfTogW1JFVFJZXSBSZXRyeWluZyBmYWlsZWQgaW1hZ2UgcHVsbDogJHtmYWlsZWRfY29weX0iCgogICAgICAgIHRhZz0kKGdyZXAgIl4ke3VyaX09IiAiJHtNQVBQSU5HX0ZJTEV9IiB8IHNlZCAncy8uKjovLycpCiAgICAgICAgY29weV9pbWFnZSAiJHtjdXJyZW50X2NvcHl9IiAiJHt0b3RhbF9jb3BpZXN9IiAiJHt1cml9IiAiJHt0YWd9IgogICAgICAgIHJjPSQ/CiAgICBkb25lCgogICAgZWNobyAiJHtQUk9HfTogW0lORk9dIEltYWdlIGxvYWQgZG9uZSIKICAgIHJldHVybiAiJHtyY30iCn0KCmlmIFtbICIke0JBU0hfU09VUkNFWzBdfSIgPSAiJHswfSIgXV07IHRoZW4KICAgIGZhaWxlZF9jb3BpZXM9KCkgIyBBcnJheSB0aGF0IHdpbGwgaW5jbHVkZSBhbGwgdGhlIGltYWdlcyB0aGF0IGZhaWxlZCB0byBiZSBwdWxsZWQKCiAgICBtb3VudF9kYXRhCgogICAgbG9hZF9pbWFnZXMKCiAgICBpZiAhIHJldHJ5X2ltYWdlczsgdGhlbgogICAgICAgIGVjaG8gIiR7UFJPR306IFtGQUlMXSAkeyNmYWlsZWRfY29waWVzW0BdfSBpbWFnZXMgd2VyZSBub3QgbG9hZGVkIHN1Y2Nlc3NmdWxseSwgYWZ0ZXIgJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiICNudW1iZXIgb2YgZmFpbGluZyBpbWFnZXMKICAgICAgICBleGl0IDEKICAgIGVsc2UKICAgICAgICBlY2hvICIke1BST0d9OiBbU1VDQ0VTU10gQWxsIGltYWdlcyB3ZXJlIGxvYWRlZCwgaW4gJCgoU0VDT05EUy1TVEFSVCkpIHNlY29uZHMiCiAgICAgICAgZXhpdCAwCiAgICBmaQpmaQo="
        }
      }
    ]
  }
}
```

##  5. <a name='StartZTP'></a>Start ZTP

Once the `siteConfig` and optionally the `policyGenTemplates` are uploaded to the Git repo where Argo CD is monitoring, we are ready to push the sync button to start the whole process. Remember that the process should require Zero Touch.



