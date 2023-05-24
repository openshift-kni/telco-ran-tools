#!/bin/bash
#
# Runs invalid parameter handling tests
#

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version 2.2.0 \
    -r 4.12.15 \
    --not-a-valid-option \
    >& command-output.txt
rc=$?

# We expect this command to return a non-zero status
if [ "${rc}" -eq 0 ]; then
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

# Check for expected error message
if ! grep -q 'Error: unknown flag: --not-a-valid-option' command-output.txt ; then
    echo "Expected error message not found in command output."
    exit 1
fi

exit 0
