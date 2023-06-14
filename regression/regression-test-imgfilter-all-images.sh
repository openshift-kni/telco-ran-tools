#!/bin/bash
#
# Runs image filter file test, filtering all images
#

source /usr/local/bin/regression-suite-common.sh

cat <<EOF > image-filter.yaml
---
patterns:
  - .*
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

# We expect this command to return a zero status
if [ "${rc}" -ne 0 ]; then
    cat command-output.txt
    echo "Expected a non-zero value, but got rc=${rc}"
    exit 1
fi

mapping_lines=$(wc -l < "${TESTFOLDER}/mapping.txt")
ai_image_lines=$(wc -l < "${TESTFOLDER}/ai-images.txt")
ocp_image_lines=$(wc -l < "${TESTFOLDER}/ocp-images.txt")
ignored_image_lines=$(wc -l < "${TESTFOLDER}/ignored-images.txt")

if [ "${mapping_lines}" -eq 0 ]; then
    cat command-output.txt
    echo "Error: mapping.txt is empty"
    exit 1
fi

if [ "${ai_image_lines}" -ne 0 ] && [ "${ocp_image_lines}" -ne 0 ]; then
    cat command-output.txt
    echo "Error: Expected ai-images.txt and ocp-images.txt to be empty"
    exit 1
fi

if [ "${ignored_image_lines}" -ne "${mapping_lines}" ]; then
    cat command-output.txt
    echo "Error: Expected ignored-images.txt to have same number of images as mapping.txt"
    exit 1
fi

exit 0
