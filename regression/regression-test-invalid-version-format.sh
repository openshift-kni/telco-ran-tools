#!/bin/bash
#
# Runs invalid parameter handling tests
#

source /usr/local/bin/regression-suite-common.sh

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version "${DEFAULT_TEST_MCE_RELEASE}" \
    -r "${DEFAULT_TEST_BAD_VERSION_FORMAT}" \
    >& command-output.txt
rc=$?

# We expect this command to return a non-zero status
if [ "${rc}" -eq 0 ]; then
    cat command-output.txt
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

# Check for expected error message
if ! grep -q "Error: Invalid release specified. X.Y.Z format expected: ${DEFAULT_TEST_BAD_VERSION_FORMAT}" command-output.txt ; then
    cat command-output.txt
    echo "Expected error message not found in command output."
    exit 1
fi

exit 0
