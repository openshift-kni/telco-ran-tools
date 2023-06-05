#!/bin/bash

set -e

WORKDIR=
function cleanup {
    if [ -f "${WORKDIR}" ]; then
        rm -rf "${WORKDIR}"
    fi
}

trap cleanup EXIT

WORKDIR=$(mktemp --tmpdir -d doc-update.XXXX)

REPODIR=$(readlink -f "$(dirname $0)/..")

#
# Manage ztp-precaching.md
#
BOOT_IGN="${REPODIR}/docs/resources/boot-beauty.ign"
DISCOVERY_IGN="${REPODIR}/docs/resources/discovery-beauty.ign"

ZTP_PRECACHING="${REPODIR}/docs/ztp-precaching.md"
TMP_ZTP_PRECACHING="${WORKDIR}/ztp-precaching.md"

for f in "${BOOT_IGN}" "${DISCOVERY_IGN}" "${ZTP_PRECACHING}"; do
    if [ ! -f "${f}" ]; then
        echo "Could not find ${f}" >&2
        exit 1
    fi
done

# Verify doc has expected data
missing_tags=0
for tag in 'json title=discovery-beauty.ign' 'json title=boot-beauty.ign' 'ignitionConfigOverride.*extract-ai' 'ignitionConfigOverride.*extract-ocp'; do
    if ! grep -q "${tag}" "${ZTP_PRECACHING}"; then
        echo "$(basename ${ZTP_PRECACHING}) missing required tag: ${tag}" >&2
        missing_tags=$((missing_tags+1))
    fi
done
if [ "${missing_tags}" -gt 0 ]; then
    exit 1
fi

# Update ztp-precaching.md with extraction script changes, if needed
echo "Updating ${ZTP_PRECACHING}"

# shellcheck disable=SC2016
sed -e '/``` json title=discovery-beauty.ign/,/```/c``` json title=discovery-beauty.ign\ninsert-discovery-beauty.ign-here\n```' \
    -e '/``` json title=boot-beauty.ign/,/```/c``` json title=boot-beauty.ign\ninsert-boot-beauty.ign-here\n```' \
    "${ZTP_PRECACHING}" > "${TMP_ZTP_PRECACHING}.sed"

awk \
    -v ai_override="ignitionConfigOverride: '$(jq -c <"${DISCOVERY_IGN}" | sed 's/\\/\\\\/g')'" \
    -v ocp_override="ignitionConfigOverride: '$(jq -c <"${BOOT_IGN}" | sed 's/\\/\\\\/g')'" \
    -v discovery_override="$(sed 's/\\/\\\\/g' ${DISCOVERY_IGN})" \
    -v boot_override="$(sed 's/\\/\\\\/g' ${BOOT_IGN})" \
    '{
        sub(/ignitionConfigOverride.*extract-ai.*/,ai_override)
        sub(/ignitionConfigOverride.*extract-ocp.*/,ocp_override)
        sub(/insert-discovery-beauty.ign-here/,discovery_override)
        sub(/insert-boot-beauty.ign-here/,boot_override)
        print
    }' \
    "${TMP_ZTP_PRECACHING}.sed" > "${TMP_ZTP_PRECACHING}.awk"

if ! cmp -s "${TMP_ZTP_PRECACHING}.awk" "${ZTP_PRECACHING}"; then
    mv "${TMP_ZTP_PRECACHING}.awk" "${ZTP_PRECACHING}"
fi

#
# Manage run-as-pod/README.md
#
RUN_AS_POD_DIR="${REPODIR}/docs/run-as-pod"
POD_DOC="${RUN_AS_POD_DIR}/README.md"
TMP_POD_DOC="${WORKDIR}/README.md"

echo "Updating ${POD_DOC}"

for yfile in "${RUN_AS_POD_DIR}"/*.yaml; do
    title=$(basename ${yfile})
    if ! grep -q "yaml title=${title}" "${POD_DOC}"; then
        continue
    fi

    echo "Processing ${title}"

    # shellcheck disable=SC2016
    sed -e '/```yaml title='"${title}"'/,/```/c```yaml title='"${title}"'\ninsert-yaml-file-here\n```' \
        "${POD_DOC}" > "${TMP_POD_DOC}.sed"

    awk \
        -v yaml_override="$(sed 's/\\/\\\\/g' ${yfile})" \
        '{
            sub(/insert-yaml-file-here/,yaml_override)
            print
        }' \
        "${TMP_POD_DOC}.sed" > "${TMP_POD_DOC}.awk"

    if ! cmp -s "${TMP_POD_DOC}.awk" "${POD_DOC}"; then
        mv "${TMP_POD_DOC}.awk" "${POD_DOC}"
    fi
done

