#!/bin/bash
#
# Runs tests for the site-config update utility
#

source /usr/local/bin/regression-suite-common.sh

#
# SC-01 - Test adding new entries
#
echo "Running: SC-01 - Test adding new entries"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          installerArgs: '["--save-partlabel","data"]'
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --cfg site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-01: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-01: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-02 - Test one entry, with two unchanged
#
echo "Running: SC-02 - Test one entry, with two unchanged"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          installerArgs: '["--save-partlabel","data"]'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --cfg site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-02: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-02: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-03 - Test dropping agent-fix-bz1964591, while keeping non-prestaging files and installerargs
#
echo "Running: SC-03 - Test dropping agent-fix-bz1964591, while keeping non-prestaging files and installerargs"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"},{"name":"var-mnt.mount","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nBefore=precache-images.service\nBindsTo=precache-images.service\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/agent-fix-bz1964591","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}},{"overwrite":true,"path":"/usr/local/bin/test-purposes","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}},{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          installerArgs: '["--testarg","keepthis"]'
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/test-purposes","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}},{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]}}'
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          installerArgs: '["--testarg","keepthis","--save-partlabel","data"]'
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --cfg site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-03: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-03: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-04 - Test adding new entries, passing in yaml via stdin
#
echo "Running: SC-04 - Test adding new entries, passing in yaml via stdin"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          installerArgs: '["--save-partlabel","data"]'
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    <site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-04: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-04: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-05 - Test passing yaml via stdin with --cfg -
#
echo "Running: SC-05 - Test passing yaml via stdin with --cfg -"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label data\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          installerArgs: '["--save-partlabel","data"]'
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label data\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --cfg - <site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-05: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-05: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-06 - Test setting a custom partition label
#
echo "Running: SC-06 - Test setting a custom partition label"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label custompartition\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
          installerArgs: '["--save-partlabel","custompartition"]'
      ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label custompartition\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --label custompartition \
    --cfg - <site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-06: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-06: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

#
# SC-07 - Test setting indentation level to 4
#
echo "Running: SC-07 - Test setting indentation level to 4"
cat <<EOF >site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "mycluster"
  namespace: "mycluster"
spec:
  # Check that comment remains in place
  baseDomain: "example.com"
  pullSecretRef:
    name: "assisted-deployment-pull-secret"
  clusters:
    - clusterName: "mycluster"
      networkType: "OVNKubernetes"
      clusterLabels:
        common: true
        sites: "mycluster"
      nodes:
        - hostName: "mynode.example.com"
          role: "master"
          bmcCredentialsName:
            name: "mynode-bmc-secret"
          bootMode: "UEFI"
          cpuset: "0-1,52-53"
          nodeNetwork:
            interfaces:
              - name: eno1
                macAddress: AA:BB:CC:DD:EE:FF
            config:
              dns-resolver:
                config:
                  #search:
                  #  - dns.example.com
                  server:
                    - 1.1.1.1
              routes:
                config:
                  - destination: 0.0.0.0/0
                    next-hop-address: 2.2.2.2
                    next-hop-interface: eno1
EOF

cat <<EOF >expected-site-config.yaml
# Subset of site-config for test purposes
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
    name: "mycluster"
    namespace: "mycluster"
spec:
    # Check that comment remains in place
    baseDomain: "example.com"
    pullSecretRef:
        name: "assisted-deployment-pull-secret"
    clusters:
        - clusterName: "mycluster"
          networkType: "OVNKubernetes"
          clusterLabels:
            common: true
            sites: "mycluster"
          nodes:
            - hostName: "mynode.example.com"
              role: "master"
              bmcCredentialsName:
                name: "mynode-bmc-secret"
              bootMode: "UEFI"
              cpuset: "0-1,52-53"
              nodeNetwork:
                interfaces:
                    - name: eno1
                      macAddress: AA:BB:CC:DD:EE:FF
                config:
                    dns-resolver:
                        config:
                            #search:
                            #  - dns.example.com
                            server:
                                - 1.1.1.1
                    routes:
                        config:
                            - destination: 0.0.0.0/0
                              next-hop-address: 2.2.2.2
                              next-hop-interface: eno1
              ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-ocp-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ocp.sh --label custompartition\nTimeoutStopSec=60\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ocp.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
              installerArgs: '["--save-partlabel","custompartition"]'
          ignitionConfigOverride: '{"ignition":{"version":"3.1.0"},"systemd":{"units":[{"name":"precache-images.service","enabled":true,"contents":"[Unit]\nDescription=Truncated for test usage\nExecStart=bash /usr/local/bin/extract-ai.sh --label custompartition\n"}]},"storage":{"files":[{"overwrite":true,"path":"/usr/local/bin/extract-ai.sh","mode":493,"user":{"name":"root"},"contents":{"source":"Truncated for test usage"}}]}}'
EOF

# Run the command, capturing the output and RC
factory-precaching-cli siteconfig \
    --testmode \
    --label custompartition \
    --indent 4 \
    --cfg - <site-config.yaml \
    > updated-site-config.yaml
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    echo "SC-07: Command returned rc=${rc}"
    exit 1
fi

if ! diff expected-site-config.yaml updated-site-config.yaml ; then
    echo "SC-07: Updated site-config.yaml doesn't match expected content."
    exit 1
fi

exit 0

