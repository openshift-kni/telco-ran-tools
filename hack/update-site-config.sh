#!/bin/bash
#
# Helper script for updating site-config with ZTP prestaging support
#

REPODIR=$(readlink -f "$(dirname $0)/..")
SITECFG=$1

BOOT_IGN="${REPODIR}/docs/resources/boot-beauty.ign"
DISCOVERY_IGN="${REPODIR}/docs/resources/discovery-beauty.ign"

if [ ! -f "${SITECFG}" ]; then
    echo "Could not find site-config file: ${SITECFG}" >&2
    exit 1
fi

for res in "${BOOT_IGN}" "${DISCOVERY_IGN}"; do
    if [ ! -f "${res}" ]; then
        echo "Could not find resource file: ${res}" >&2
        exit 1
    fi
done

function update_existing {
    # shellcheck disable=SC2016
    awk -i inplace '
        {
            sub(/ignitionConfigOverride.*extract-ai.*/,ai_override)
            sub(/ignitionConfigOverride.*extract-ocp.*/,ocp_override)
            print
        }' \
        ai_override="ignitionConfigOverride: '$(jq -c <"${DISCOVERY_IGN}" | sed 's/\\/\\\\/g')'" \
        ocp_override="ignitionConfigOverride: '$(jq -c <"${BOOT_IGN}" | sed 's/\\/\\\\/g')'" \
    "${SITECFG}"
}

function insert_support {
    if grep -q -e installerArgs -e ignitionConfigOverride "${SITECFG}"; then
        echo "Error: site-config already has installerArgs or ignitionConfigOverride and cannot be updated automatically" >&2
        exit 1
    fi

    # shellcheck disable=SC2016
    awk -i inplace '
        {
            if ($0 ~ /- clusterName:/) {
                print
                sub(/- clusterName:.*/,ai_override)
                print "  " $0
            } else if ($0 ~ /clusterName:/) {
                print
                sub(/clusterName:.*/,ai_override)
                print
            } else if ($0 ~ /- hostName:/) {
                print
                sub(/- hostName:.*/,installer_args)
                print "  " $0
                sub(/installerArgs:.*/,ocp_override)
                print "  " $0
            } else if ($0 ~ /hostName:/) {
                print
                sub(/hostName:.*/,installer_args)
                print
                sub(/installerArgs:.*/,ocp_override)
                print
            } else {
                print
            }
        }' \
        ai_override="ignitionConfigOverride: '$(jq -c <"${DISCOVERY_IGN}" | sed 's/\\/\\\\/g')'" \
        ocp_override="ignitionConfigOverride: '$(jq -c <"${BOOT_IGN}" | sed 's/\\/\\\\/g')'" \
        installer_args="installerArgs: '[\"--save-partlabel\", \"data\"]'" \
    "${SITECFG}"
}

if grep -q 'ignitionConfigOverride.*extract-ai' "${SITECFG}"; then
    update_existing
else
    insert_support
fi

