#!/bin/bash
#
# Runs image filter file test, filtering with regex with no matches
#

source /usr/local/bin/regression-suite-common.sh

cat <<EOF > image-filter.yaml
---
patterns:
  - ^qua\b
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
downloaded_image_lines=$(sort -u "${TESTFOLDER}/ai-images.txt" "${TESTFOLDER}/ocp-images.txt" | wc -l)
ignored_image_lines=$(wc -l < "${TESTFOLDER}/ignored-images.txt")

if [ "${ignored_image_lines}" -ne 0 ]; then
    cat command-output.txt
    echo "Expected no images to be filtered."
    exit 1
fi

if [ "${mapping_lines}" -eq 0 ]; then
    cat command-output.txt
    echo "Error: mapping.txt is empty"
    exit 1
fi

if [ "${downloaded_image_lines}" -eq 0 ]; then
    cat command-output.txt
    echo "Error: Expected ai-images.txt and ocp-images.txt to have images"
    exit 1
fi

if [ "$((downloaded_image_lines+ignored_image_lines))" -ne "${mapping_lines}" ]; then
    cat command-output.txt
    echo "Error: Expected sum of downloaded and ignored images to match mapping.txt entries"
    exit 1
fi

exit 0
