#!/bin/bash
#
# Runs invalid parameter handling tests
#

source /usr/local/bin/regression-suite-functions.sh

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version 2.2.0 \
    -r 4.12.15 \
    --generate-imageset \
    >& command-output.txt
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    cat command-output.txt
    echo "Command returned rc=${rc}"
    exit 1
fi

# Verify content of imageset, and that it is the only file in the TESTFOLDER directory

if [ ! -f "${TESTFOLDER}/imageset.yaml" ]; then
    echo "Could not find ${TESTFOLDER}/imageset.yaml"
    exit 1
fi

cat <<EOF >expected-imageset.yaml

---
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
mirror:
  platform:
    channels:
    - name: stable-4.12
      minVersion: 4.12.15
      maxVersion: 4.12.15
  additionalImages:
#
# Example operators specification:
#
#  operators:
#    - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.11
#      full: true
#      packages:
#        - name: ptp-operator
#          channels:
#            - name: 'stable'
#        - name: sriov-network-operator
#          channels:
#            - name: 'stable'
#        - name: cluster-logging
#          channels:
#            - name: 'stable'
  operators:
    - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.12
      packages:
        - name: multicluster-engine
          channels:
            - name: 'stable-2.2'
              minVersion: 2.2.0
              maxVersion: 2.2.0
EOF

if ! diff expected-imageset.yaml "${TESTFOLDER}/imageset.yaml" ; then
    echo "Generated imageset.yaml doesn't match expected content."
    exit 1
fi

numfiles=$(find "${TESTFOLDER}" -type f)
if [ "${numfiles}" -ne 1 ]; then
    echo "Expected to find 1 file in ${TESTFOLDER}, but found ${numfiles}"
    exit 1
fi

exit 0
