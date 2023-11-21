# factory-precaching-cli: Running as a Pod #

- [factory-precaching-cli: Running as a Pod](#factory-precaching-cli-running-as-a-pod)
  - [Overview](#overview)
  - [Pre-requisites](#pre-requisites)
  - [Creating namespace and security resources](#creating-namespace-and-security-resources)
    - [010-scc.yaml](#010-sccyaml)
    - [020-cluster-role.yaml](#020-cluster-roleyaml)
    - [030-namespace.yaml](#030-namespaceyaml)
    - [040-service-account.yaml](#040-service-accountyaml)
    - [050-cluster-role-binding.yaml](#050-cluster-role-bindingyaml)
    - [060-storage-class.yaml](#060-storage-classyaml)
  - [Creating PV, PVC, and Config Map Resources](#creating-pv-pvc-and-config-map-resources)
    - [070-pv.yaml](#070-pvyaml)
    - [080-pvc.yaml](#080-pvcyaml)
    - [090-config-map.yaml](#090-config-mapyaml)
  - [Running Download](#running-download)
    - [100-download.yaml](#100-downloadyaml)
    - [100-download-with-pattern-filter.yaml](#100-download-with-pattern-filteryaml)
    - [100-download-with-imageset.yaml](#100-download-with-imagesetyaml)
  - [Running Regression Test Suite](#running-regression-test-suite)
    - [110-regression-suite.yaml](#110-regression-suiteyaml)
  - [Cleanup](#cleanup)

## Overview ##

For a full description of the download command, please see the [downloading.md](../downloading.md) document. There you will find information on how to use the download command and its features.

This document is a guide for running the `factory-precaching-cli` tool in a pod on a running SNO. The configuration examples provided can be customized to suit your needs. In this example, we create the following custom resources:

- Security Context Constraint (SCC) with minimal privileges and capabilities: `telco-ran-tools-scc`
- Cluster Role to use the custom SCC: `telco-ran-tool-cluster-role`
- Namespace: `telco-ran-tools`
- Service Account (SA): `telco-ran-tools-user`
- Cluster Role Binding, binding the custom role to the SA: `telco-ran-tools-crb`
- Storage Class: `telco-ran-tools-storage-class`
- Persistent Volume (PV), to provide access to the prestaging partition (with default label `data`): `telco-ran-tools-storage`
- Persistent Volume Claim (PVC): `telco-ran-tools-storage-pvc`
- Config Map (CM), optionally providing `image-filters.yaml` and a predefined/customizable `imageset.yaml` example: `prestaging-data`
- Example pods for running downloads or running the regression test suite

## Pre-requisites ##

- An unmounted partition has already been created on the SNO, with an `xfs` filesystem created, large enough for the downloaded images and associated files.

## Creating namespace and security resources ##

### 010-scc.yaml ###

The custom SCC is configured to allow the pod access to the host network and prestaging partition, as well as the config map and pull secrets. In addition, it allows the pod to run as the root user for writing files to the prestaging partition.

```yaml title=010-scc.yaml
---
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  annotations:
    kubernetes.io/description: Custom SCC for telco-ran-tools
  name: telco-ran-tools-scc
#
# The telco-ran-tools factory-precaching-cli tool requires:
# - network access
# - write-access to disk partition
#
# Additionally, the pod requires permission to mount the disk and configmap as volumes
#
allowHostDirVolumePlugin: true
allowHostIPC: false
allowHostNetwork: true
allowHostPID: false
allowHostPorts: false
allowPrivilegeEscalation: false
allowPrivilegedContainer: false
allowedCapabilities: null
defaultAddCapabilities: null
groups: []
priority: null
readOnlyRootFilesystem: false
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: MustRunAs
fsGroup:
  type: RunAsAny
supplementalGroups:
  type: RunAsAny
users: []
volumes:
- configMap
- persistentVolumeClaim
- secret
```

### 020-cluster-role.yaml ###

```yaml title=020-cluster-role.yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: telco-ran-tool-cluster-role
rules:
- apiGroups:
  - security.openshift.io
  # Use the custom SCC we've created
  resourceNames:
  - telco-ran-tools-scc
  resources:
  - securitycontextconstraints
  verbs:
  - use
  resources:
  - securitycontextconstraints
  verbs:
  - use
```

### 030-namespace.yaml ###

```yaml title=030-namespace.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: telco-ran-tools
```

### 040-service-account.yaml ###

```yaml title=040-service-account.yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: telco-ran-tools-user
  namespace: telco-ran-tools
```

### 050-cluster-role-binding.yaml ###

```yaml title=050-cluster-role-binding.yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: telco-ran-tools-crb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: telco-ran-tool-cluster-role
subjects:
  - kind: ServiceAccount
    name: telco-ran-tools-user
    namespace: telco-ran-tools
```

### 060-storage-class.yaml ###

```yaml title=060-storage-class.yaml
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: telco-ran-tools-storage-class
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
```

## Creating PV, PVC, and Config Map Resources ##

### 070-pv.yaml ###

The `telco-ran-tools-storage` PV provides access to the prestaging disk partition on the SNO. In this example, we are using a partition labeled `data`, with a 250G xfs filesystem.

```yaml title=070-pv.yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: telco-ran-tools-storage
spec:
  capacity:
    storage: 250Gi
  volumeMode: Filesystem
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Delete
  storageClassName: telco-ran-tools-storage-class
  local:
    path: /dev/disk/by-partlabel/data
    fsType: xfs
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: node-role.kubernetes.io/master
              operator: In
              values:
                - ""
```

### 080-pvc.yaml ###

```yaml title=080-pvc.yaml
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: telco-ran-tools-storage-pvc
  namespace: telco-ran-tools
spec:
  accessModes:
  - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 250Gi
  storageClassName: telco-ran-tools-storage-class
```

### 090-config-map.yaml ###

The `prestaging-data` config map provides a means of passing content to the `factory-precaching-cli download` command as file arguments.
In this example, we have an `image-filters.yaml` with a set of regex patterns to filter images we don't need to download for prestaging.
Additionally, we have an example `imageset.yaml` that can be customized and passed to the `download` command if we want to modify the generated
`imageset.yaml` with custom catalog references or operators.

```yaml title=090-config-map.yaml
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: prestaging-data
  namespace: telco-ran-tools
data:
  #
  # Example image-filters.yaml:
  # This list of regex patterns can be customized and passed to the
  # factory-precaching-cli download command in the job configuration
  #
  image-filters.yaml: |
    ---
    patterns:
      - alibaba
      - aws
      - azure
      - cluster-samples-operator
      - gcp
      - ibm
      - kubevirt
      - libvirt
      - manila
      - nfs
      - nutanix
      - openstack
      - ovirt
      - sdn
      - tests
      - thanos
      - vsphere
  #
  # Example imageset.yaml:
  # Combined with the --skip-imageset option of the factory-precaching-cli download
  # command, a predefined imageset config can be customized as needed. The following
  # is an example imageset generated by the download command with the following options:
  #     --release 4.12.9
  #     --mce-version 2.2.3
  #     --acm-version 2.7.3
  #     --du-profile
  #     --generate-imageset
  #
  imageset.yaml: |
    ---
    apiVersion: mirror.openshift.io/v1alpha2
    kind: ImageSetConfiguration
    mirror:
      platform:
        channels:
        - name: stable-4.12
          minVersion: 4.12.9
          maxVersion: 4.12.9
      additionalImages:
      operators:
        - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.12
          packages:
            - name: multicluster-engine
              channels:
                - name: 'stable-2.4'
                - name: 'stable-2.2'
                  minVersion: 2.2.3
                  maxVersion: 2.2.3
            - name: advanced-cluster-management
              channels:
                - name: 'release-2.9'
                - name: 'release-2.7'
                  minVersion: 2.7.3
                  maxVersion: 2.7.3
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
        - catalog: registry.redhat.io/redhat/certified-operator-index:v4.12
          packages:
            - name: sriov-fec
              channels:
                - name: 'stable'
```

## Running Download ##

### 100-download.yaml ###

In this example, we're running the `download` command without specifying image filters or a customized `imageset.yaml`, but are limiting
parallelization to a single worker thread in order to reduce CPU impact.

```yaml title=100-download.yaml
---
apiVersion: batch/v1
kind: Job
metadata:
  name: telco-ran-tools-download
  namespace: telco-ran-tools
spec:
  backoffLimit: 0
  template:
    metadata:
      name: telco-ran-tools-download
      namespace: telco-ran-tools
    spec:
      containers:
      - name: telco-ran-tools-download
        image: quay.io/openshift-kni/telco-ran-tools:latest
        #
        # Call factory-precaching-cli with desired options:
        # - To minimize CPU, use "--parallel 1" to run with a single worker
        #
        command: ["factory-precaching-cli", "download",
                    "-f", "/prestaging",
                    "-r", "4.12.5",
                    "--du-profile",
                    "--mce-version", "2.2.0",
                    "--acm-version", "2.7.0",
                    "--rm-stale",
                    "--parallel", "1"
                 ]
        securityContext:
          runAsUser: 0
        tty: true
        volumeMounts:
        # Mount the pull-secrets in the location referenced by oc-mirror
        - name: pull-secret
          mountPath: /root/.docker/config.json
        # Mount the data partition
        - name: storage
          mountPath: /prestaging
      hostNetwork: true
      restartPolicy: Never
      securityContext: {}
      serviceAccountName: telco-ran-tools-user
      volumes:
        # Use the pull-secrets from the SNO
        - name: pull-secret
          hostPath:
            path: /var/lib/kubelet/config.json
            type: File
        # Access the data partition
        - name: storage
          persistentVolumeClaim:
            claimName: telco-ran-tools-storage-pvc
```

### 100-download-with-pattern-filter.yaml ###

In this example, we're accessing the `prestaging-data` config map to pass in the `image-filters.yaml` to reduce the list of images that are downloaded. In addition,
by not specifying the `parallel` option, the `download` will run parallel threads up to tje default maximum of 80% of the processors available to the pod.

```yaml title=100-download-with-pattern-filter.yaml
---
apiVersion: batch/v1
kind: Job
metadata:
  name: telco-ran-tools-download
  namespace: telco-ran-tools
spec:
  backoffLimit: 0
  template:
    metadata:
      name: telco-ran-tools-download
      namespace: telco-ran-tools
    spec:
      containers:
      - name: telco-ran-tools-download
        image: quay.io/openshift-kni/telco-ran-tools:latest
        #
        # Call factory-precaching-cli with desired options:
        # - To minimize CPU, use "--parallel 1" to run with a single worker
        # - The "--filter" option is used in conjunction with the set of image filter
        #   patterns defined in the prestaging-data configmap
        #
        command: ["factory-precaching-cli", "download",
                    "-f", "/prestaging",
                    "--filter", "/prestaging/image-filters.yaml",
                    "-r", "4.12.15",
                    "--du-profile",
                    "--mce-version", "2.2.3",
                    "--acm-version", "2.7.3",
                    "--rm-stale"
                 ]
        securityContext:
          runAsUser: 0
        tty: true
        volumeMounts:
        # Mount the pull-secrets in the location referenced by oc-mirror
        - name: pull-secret
          mountPath: /root/.docker/config.json
        # Mount the data partition
        - name: storage
          mountPath: /prestaging
        # Mount the image-filters.yaml from the configmap
        # (path isn't important, as long as it matches the --filter option specified)
        - name: prestaging-data-volume
          mountPath: /prestaging/image-filters.yaml
          subPath: image-filters.yaml
      hostNetwork: true
      restartPolicy: Never
      securityContext: {}
      serviceAccountName: telco-ran-tools-user
      volumes:
        # Use the pull-secrets from the SNO
        - name: pull-secret
          hostPath:
            path: /var/lib/kubelet/config.json
            type: File
        # Access the data partition
        - name: storage
          persistentVolumeClaim:
            claimName: telco-ran-tools-storage-pvc
        # Access the prestaging-data configmap
        - name: prestaging-data-volume
          configMap:
            name: prestaging-data
```

### 100-download-with-imageset.yaml ###

In this example, in addition to passing in the `image-filters.yaml` and setting maximum parallel threads to 80, we're also referencing the `imageset.yaml` from the config map, rather than having `download` generate it.

```yaml title=100-download-with-imageset.yaml
---
apiVersion: batch/v1
kind: Job
metadata:
  name: telco-ran-tools-download
  namespace: telco-ran-tools
spec:
  backoffLimit: 0
  template:
    metadata:
      name: telco-ran-tools-download
      namespace: telco-ran-tools
    spec:
      containers:
      - name: telco-ran-tools-download
        image: quay.io/openshift-kni/telco-ran-tools:latest
        #
        # Call factory-precaching-cli with desired options:
        # - To minimize CPU, use "--parallel 1" to run with a single worker
        # - The "--skip-imageset" option is used in conjunction with the predefined
        #   imageset.yaml from the prestaging-data configmap
        # - The "--filter" option is used in conjunction with the set of image filter
        #   patterns defined in the prestaging-data configmap
        #
        command: ["factory-precaching-cli", "download",
                    "--folder", "/prestaging",
                    "--filter", "/prestaging/image-filters.yaml",
                    "--release", "4.12.9",
                    "--du-profile",
                    "--mce-version", "2.2.3",
                    "--acm-version", "2.7.3",
                    "--rm-stale",
                    "--parallel", "80",
                    "--skip-imageset"
                 ]
        securityContext:
          runAsUser: 0
        tty: true
        volumeMounts:
        # Mount the pull-secrets in the location referenced by oc-mirror
        - name: pull-secret
          mountPath: /root/.docker/config.json
        # Mount the data partition
        - name: storage
          mountPath: /prestaging
        # Mount the imageset.yaml from the configmap
        # (must be located in the path specified by the --folder option and named imageset.yaml)
        - name: prestaging-data-volume
          mountPath: /prestaging/imageset.yaml
          subPath: imageset.yaml
        # Mount the image-filters.yaml from the configmap
        # (path isn't important, as long as it matches the --filter option specified)
        - name: prestaging-data-volume
          mountPath: /prestaging/image-filters.yaml
          subPath: image-filters.yaml
      hostNetwork: true
      restartPolicy: Never
      securityContext: {}
      serviceAccountName: telco-ran-tools-user
      volumes:
        # Use the pull-secrets from the SNO
        - name: pull-secret
          hostPath:
            path: /var/lib/kubelet/config.json
            type: File
        # Access the data partition
        - name: storage
          persistentVolumeClaim:
            claimName: telco-ran-tools-storage-pvc
        # Access the prestaging-data configmap
        - name: prestaging-data-volume
          configMap:
            name: prestaging-data
```

## Running Regression Test Suite ##

The regression test suite is built into the `quay.io/openshift-kni/telco-ran-tools:latest` image. It requires host network and pull secret access,
but does not use the prestaging partition or `prestaging-data` config map, and does not need to run as `root`. Creating this job will launch a pod
that runs `regression-suite.sh`, and the results can be viewed in the pod's logs using the `oc log` command.

### 110-regression-suite.yaml ###

```yaml title=110-regression-suite.yaml
---
apiVersion: batch/v1
kind: Job
metadata:
  name: telco-ran-tools-regression
  namespace: telco-ran-tools
spec:
  backoffLimit: 0
  template:
    metadata:
      name: telco-ran-tools-regression
      namespace: telco-ran-tools
    spec:
      containers:
      - name: telco-ran-tools-regression
        image: quay.io/openshift-kni/telco-ran-tools:latest
        #
        # Run regression suite
        #
        command: ["regression-suite.sh"]
        tty: true
        volumeMounts:
        # Mount the pull-secrets in the location referenced by oc-mirror
        - name: pull-secret
          mountPath: /root/.docker/config.json
      hostNetwork: true
      restartPolicy: Never
      securityContext: {}
      serviceAccountName: telco-ran-tools-user
      volumes:
        # Use the pull-secrets from the SNO
        - name: pull-secret
          hostPath:
            path: /var/lib/kubelet/config.json
            type: File
```

## Cleanup ##

Because most of these resources are created within our custom namespace, most of the cleanup is done by deleting the namespace. Those resources that are not namespaced must be deleted individually.

```console
oc delete ns telco-ran-tools
oc delete storageclasses.storage.k8s.io telco-ran-tools-storage-class
oc delete clusterrolebindings telco-ran-tools-crb
oc delete pv telco-ran-tools-storage
oc delete clusterrole telco-ran-tool-cluster-role
oc delete scc telco-ran-tools-scc
```
