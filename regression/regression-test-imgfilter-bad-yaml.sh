#!/bin/bash
#
# Runs image filter file test, with bad yaml
#

source /usr/local/bin/regression-suite-common.sh

cat <<EOF > image-filter.yaml
---
patterns:
  - [bad-yaml
EOF

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version "${DEFAULT_TEST_MCE_RELEASE}" \
    -r "${DEFAULT_TEST_RELEASE}" \
    --filter image-filter.yaml \
    >& command-output.txt
rc=$?

# We expect this command to return a non-zero status
if [ "${rc}" -eq 0 ]; then
    cat command-output.txt
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

# Check for expected error message
if ! grep -q "Error: yaml: line [0-9]*: did not find expected" command-output.txt ; then
    cat command-output.txt
    echo "Expected error message not found in command output."
    exit 1
fi

exit 0
