#!/bin/bash
#
# Runs invalid parameter handling tests
#

source /usr/local/bin/regression-suite-common.sh

# Create a fake image file for stale cleanup test
echo "Test image" > "${TESTFOLDER}"/not-a-real-image@sha256_1234567890123456789012345678901234567890123456789012345678901234.tgz

# Run the command, capturing the output and RC
factory-precaching-cli download \
    --testmode \
    -f "${TESTFOLDER}" \
    --mce-version "${DEFAULT_TEST_MCE_RELEASE}" \
    -r "${DEFAULT_TEST_RELEASE}" \
    >& command-output.txt
rc=$?

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    cat command-output.txt
    echo "Command returned rc=${rc}"
    exit 1
fi

# Verify all expected image files exist
if ! verify_downloaded_files ; then
    exit 1
fi

# We expect to see our stale file remaining
if check_for_stale_image_files ; then
    echo "Error: No stale files detected"
    exit 1
fi

exit 0
