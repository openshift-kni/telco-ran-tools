#!/bin/bash

#
# Check mapping.txt to ensure all files are downloaded and in either ai-images.txt or ocp-images.txt
#
function verify_downloaded_files {
    local missing=0
    local verified=0

    local mapping
    local img
    local imgfile

    for mapping in $(<"${TESTFOLDER}/mapping.txt"); do
        img=$(basename ${mapping/=*})

        # Image must be in either ai-images.txt or ocp-images.txt
        if ! grep -q "/${img}$" "${TESTFOLDER}/ai-images.txt" "${TESTFOLDER}/ocp-images.txt"; then
            echo "Missing from images files: ${img}"
            missing=$((missing+1))
            continue
        else
            verified=$((verified+1))
        fi

        # Image filename has the : replaced with _, with a .tgz suffix
        imgfile="${TESTFOLDER}/${img/:/_}.tgz"
        if [ ! -f "${imgfile}" ]; then
            echo "File not found: ${imgfile}"
            missing=$((missing+1))
            continue
        fi
    done

    if [ "${verified}" -eq 0 ]; then
        echo "Verification failed due to missing files in ${TESTFOLDER}/mapping.txt"
        return 1
    fi

    if [ "${missing}" -gt 0 ]; then
        echo "Verification failed due to missing files or filenames"
        return 1
    fi

    return 0
}

#
# Check for stale image files
#
function check_for_stale_image_files {
    local -a expected_files
    local mapping
    local img
    local imgfile

    for mapping in $(<"${TESTFOLDER}/mapping.txt"); do
        img=$(basename ${mapping/=*})

        # Image filename has the : replaced with _, with a .tgz suffix
        expected_files+=("${TESTFOLDER}/${img/:/_}.tgz")
    done

    local stale=0

    for imgfile in "${TESTFOLDER}"/*.tgz; do
        if [[ ! "${expected_files[*]}" =~ ${imgfile} ]]; then
            echo "Stale file found: ${imgfile}"
            stale=$((stale+1))
        fi
    done

    return "${stale}"
}

