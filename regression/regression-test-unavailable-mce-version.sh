#!/bin/bash
#
# Runs invalid parameter handling tests
#

source /usr/local/bin/regression-suite-common.sh

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version "${DEFAULT_TEST_UNAVAILABLE_VERSION}" \
    -r "${DEFAULT_TEST_RELEASE}" \
    >& command-output.txt
rc=$?

# We expect this command to return a non-zero status
if [ "${rc}" -eq 0 ]; then
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

# Check for expected error message
if ! grep -q "multicluster-engine version ${DEFAULT_TEST_UNAVAILABLE_VERSION} not found in channel" command-output.txt || \
    ! grep -q 'Version checks failed for 1 operator' command-output.txt ; then
    echo "Expected error message not found in command output."
    exit 1
fi

exit 0
