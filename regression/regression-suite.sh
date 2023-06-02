#!/bin/bash
#
# Runs a set of tests within the telco-ran-tools container
#

PROG=$(basename "$0")
SCRIPTDIR=$(dirname $(readlink -f "$0"))

# Location of regression tests
declare TESTDIR="${SCRIPTDIR}/regression-tests"

# Default folder
declare FOLDER="/mnt"

#
# Usage
#
function usage {
    cat <<EOF
This script runs a series of regression tests within the telco-ran-tools container.

Usage: ${PROG}
Options:
EOF
    exit 1
}

#
# Display a separator
#
function separator {
    printf '#%.0s' {1..80}; printf '\n'
}

#
# Process cmdline arguments
#

longopts=(
    "help"
    "folder:"
    "test:"
)

longopts_str=$(IFS=,; echo "${longopts[*]}")

if ! OPTS=$(getopt -o "hf:" --long "${longopts_str}" --name "$0" -- "$@"); then
    usage
    exit 1
fi

eval set -- "${OPTS}"

declare -a RUN_TESTS=()

while :; do
    case "$1" in
        -f|--folder)
            FOLDER="${2}"
            shift 2
            ;;
        --test)
            if [ -f "${TESTDIR}/regression-test-${2}.sh" ]; then
                RUN_TESTS+=("${TESTDIR}/regression-test-${2}.sh")
            elif [ -f "${TESTDIR}/${2}" ]; then
                RUN_TESTS+=("${TESTDIR}/${2}")
            else
                echo "Could not find test: ${2}" >&1
                exit 1
            fi
            shift 2
            ;;
        --)
            shift
            break
            ;;
        -h|--help)
            usage
            ;;
        *)
            usage
            ;;
    esac
done

# Make TESTFOLDER available to tests
export TESTFOLDER="${FOLDER}/testsuite"

if [ -d "${TESTFOLDER}" ]; then
    echo "Content from previous test run exists. Please delete: ${TESTFOLDER}" >&2
    exit 1
fi

# Run the regression tests
declare -A results

if [ "${#RUN_TESTS[@]}" -eq 0 ]; then
    RUN_TESTS=("${TESTDIR}"/regression-test-*.sh)
fi

for testscript in "${RUN_TESTS[@]}"; do
    if ! mkdir "${TESTFOLDER}" ; then
        echo "Unable to create ${TESTFOLDER}" >&2
        exit 1
    fi

    WORKDIR=$(mktemp -d)
    cd "${WORKDIR}" || exit 1

    name=$(echo "${testscript}" | sed -r 's#.*/regression-test-(.*)\.sh$#\1#')
    separator
    echo "$(date): Running test: ${name}"

    ${testscript}
    results["${name}"]=$?

    cd /
    rm -rf "${WORKDIR}"
    rm -rf "${TESTFOLDER}"
done

separator

# Report results
echo -e '\nTest Results:\n'
printf '%-40s  %-10s\n' 'Test Name' 'Status'
printf '=%.0s' {1..40} ; printf '  '; printf '=%.0s' {1..10}; printf '\n'

total_passed=0
total_failed=0

mapfile -t sorted_names < <( IFS=$'\n'; sort -u <<<"${!results[*]}" )
for name in "${sorted_names[@]}"; do
    if [ "${results[${name}]}" -eq 0 ]; then
        status="PASSED"
        total_passed=$((total_passed+1))
    else
        status="FAILED"
        total_failed=$((total_failed+1))
    fi
    printf '%-40s  %-10s\n' "${name}" "${status}"
done

printf '\n'

printf '%-20s %3d\n' 'Number of tests run:' ${#results[@]}
printf '%-20s %3d\n' 'Total passed:' ${total_passed}
printf '%-20s %3d\n' 'Total failed:' ${total_failed}

if [ "${total_failed}" -gt 0 ]; then
    exit 1
fi

exit 0

