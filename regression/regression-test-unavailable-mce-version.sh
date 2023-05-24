#!/bin/bash
#
# Runs invalid parameter handling tests
#

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version 99.99.99 \
    -r 4.12.15 \
    >& command-output.txt
rc=$?

# We expect this command to return a non-zero status
if [ "${rc}" -eq 0 ]; then
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

# Check for expected error message
if ! grep -q 'multicluster-engine version 99.99.99 not found in channel' command-output.txt || \
    ! grep -q 'Version checks failed for 1 operator' command-output.txt ; then
    echo "Expected error message not found in command output."
    exit 1
fi

exit 0
