#!/bin/bash
#
# Runs test for specifying catalogs
#

source /usr/local/bin/regression-suite-common.sh

TEST_CATALOG_REDHAT_OPERATORS="nonexistent.catalog:5000/redhat/redhat-operator-index:v99.99"
TEST_CATALOG_CERTIFIED_OPERATORS="nonexistent.catalog:5000/redhat/certified-operator-index:v99.99"

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version "${DEFAULT_TEST_MCE_RELEASE}" \
    -r "${DEFAULT_TEST_RELEASE}" \
    --du-profile \
    --acm-version "${DEFAULT_TEST_ACM_RELEASE}" \
    --catalog-redhat-operators "${TEST_CATALOG_REDHAT_OPERATORS}" \
    --catalog-certified-operators "${TEST_CATALOG_CERTIFIED_OPERATORS}" \
    --generate-imageset \
    >& command-output.txt
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    cat command-output.txt
    echo "Command returned rc=${rc}"
    exit 1
fi

# Validate imageset.yaml
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
    - name: stable-${DEFAULT_TEST_RELEASE_Y}
      minVersion: ${DEFAULT_TEST_RELEASE}
      maxVersion: ${DEFAULT_TEST_RELEASE}
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
    - catalog: ${TEST_CATALOG_REDHAT_OPERATORS}
      packages:
        - name: multicluster-engine
          channels:
            - name: 'stable-2.3'
            - name: 'stable-${DEFAULT_TEST_MCE_RELEASE_Y}'
              minVersion: ${DEFAULT_TEST_MCE_RELEASE}
              maxVersion: ${DEFAULT_TEST_MCE_RELEASE}
        - name: advanced-cluster-management
          channels:
            - name: 'release-${DEFAULT_TEST_ACM_RELEASE_Y}'
              minVersion: ${DEFAULT_TEST_ACM_RELEASE}
              maxVersion: ${DEFAULT_TEST_ACM_RELEASE}
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
    - catalog: ${TEST_CATALOG_CERTIFIED_OPERATORS}
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
EOF

if ! diff expected-imageset.yaml "${TESTFOLDER}/imageset.yaml" ; then
    echo "Generated imageset.yaml doesn't match expected content."
    exit 1
fi

exit 0
