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

# Update ztp-precaching.md with extraction script changes, if needed
echo "Updating ${ZTP_PRECACHING}"

# shellcheck disable=SC2016
sed -e '/``` { .json title=discovery-beauty.ign }/,/```/c``` { .json title=discovery-beauty.ign }\ninsert-discovery-beauty.ign-here\n```' \
    -e '/``` { .json title=boot-beauty.ign }/,/```/c``` { .json title=boot-beauty.ign }\ninsert-boot-beauty.ign-here\n```' \
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
