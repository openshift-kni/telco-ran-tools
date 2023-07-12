#!/bin/bash
#
# Helper script for updating site-config with ZTP prestaging support
#

#
# Check required tools
#
REQUIRED_TOOLS=(
    jq
    yq
)
MISSING_TOOLS=()

for tool in "${REQUIRED_TOOLS[@]}"; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
        MISSING_TOOLS+=( "${tool}" )
    fi
done

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo "The following required tools are missing:" >&2
    echo "${MISSING_TOOLS[@]}" >&2
    exit 1
fi

#
# Constants
#
declare CLUSTER_IGNITION='.spec.clusters[0].ignitionConfigOverride'
declare NODE_IGNITION='.spec.clusters[0].nodes[0].ignitionConfigOverride'
declare NODE_INSTALLER_ARGS='.spec.clusters[0].nodes[0].installerArgs'

#
# Usage
#
function usage {
    cat <<EOF >&2
Usage: $(basename $0) [ --label <partition-label> ] [ --resources-dir <dir> ] --site-config <file>

Parameters:
    -l | --label <partition-label>   - Specify custom prestaging partition label (default: data)
    -r | --resources-dir <dir>       - Specify dir where boot-beauty.ign and discovery-beauty.ign are located
    -s | --site-config <file>        - Specify site-config filename
EOF

    exit 1
}

#
# Strip prestaging entries from ignition config, including deprecated ones
#
function delete_prestaging_entries {
    jq -cM '
        if type == "null" then empty else

        if .systemd?.units then
            # Delete prestaging service-units, including legacy content
            del( .systemd.units[]
                | select(
                    .name == "precache-images.service"
                    or .name == "precache-ocp-images.service"
                    or ( .name == "var-mnt.mount" and (.contents | test ("BindsTo=precache")) )
                    )
                )
        else . end

        | if .storage?.files then
            # Delete prestaging files, including legacy content
            del( .storage.files[]
                | select(
                    .path == "/usr/local/bin/extract-ai.sh"
                    or .path == "/usr/local/bin/extract-ocp.sh"
                    or .path == "/usr/local/bin/agent-fix-bz1964591"
                    )
                )
        else . end

        # Delete empty objects that remain
        | walk(
            if type == "object" then
                with_entries(if .value == {} or .value == [] then empty else . end)
            else .  end)

        # Delete entries where the only thing left is the "ignition" data
        | del(select(keys == ["ignition"]))

        | if type == "null" then empty else . end

        end
    '
}

#
# Merge ignition config
#
function merge_ignition_config {
    local field="${1}"
    local ign="${2}"

    (
        yq "${field}" "${SITECFG}" | delete_prestaging_entries
        sed -e "s/--label data/--label ${PARTITION_LABEL}/g" "${ign}"
    ) \
    | jq -csM '
        if length == 1 then
            # Only one element means there were no entries in the original
            # site-config (or all have been pruned), so just return the prestaging config
            .[0]
        else
            # Append the prestaging systemd units to the original site-config
            if .[0].systemd then
                if .[0].systemd.units then
                    .[0].systemd.units += .[1].systemd.units
                else
                    .[0].systemd.units = .[1].systemd.units
                end
            else
                .[0].systemd = .[1].systemd
            end
            # Append the prestaging storage files to the original site-config
            | if .[0].storage then
                if .[0].storage.files then
                    .[0].storage.files += .[1].storage.files
                else
                    .[0].storage.files = .[1].storage.files
                end
            else
                .[0].storage = .[1].storage
            end
            | .[0]
        end
    '
}

#
# Update existing installer args
#
function update_installer_args {
    local installer_args='"--save-partlabel","'"${PARTITION_LABEL}"'"'
    if ! yq "${NODE_INSTALLER_ARGS}" "${SITECFG}" | jq -cM | grep "${installer_args}"; then
        yq "${NODE_INSTALLER_ARGS}" "${SITECFG}" \
            | jq -cM --arg partlabel "${PARTITION_LABEL}" '. + ["--save-partlabel", $partlabel]'
    fi
}

#
# Process cmdline arguments
#

longopts=(
    "help"
    "label:"
    "resources-dir:"
    "site-config:"
)

longopts_str=$(IFS=,; echo "${longopts[*]}")

if ! OPTS=$(getopt -o "hl:r:s:" --long "${longopts_str}" --name "$0" -- "$@"); then
    usage
    exit 1
fi

eval set -- "${OPTS}"

PARTITION_LABEL="data"

while :; do
    case "$1" in
        -l|--label)
            PARTITION_LABEL="${2}"
            shift 2
            ;;
        -r|--resources-dir)
            RESOURCES_DIR="${2}"
            shift 2
            ;;
        -s|--site-config)
            SITECFG="${2}"
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

if [ -z "${RESOURCES_DIR}" ]; then
    REPODIR=$(readlink -f "$(dirname $0)/..")
    RESOURCES_DIR="${REPODIR}/docs/resources"
fi

if [ -z "${SITECFG}" ]; then
    SITECFG=$1
fi

BOOT_IGN="${RESOURCES_DIR}/boot-beauty.ign"
DISCOVERY_IGN="${RESOURCES_DIR}/discovery-beauty.ign"

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

#
# Merge the site-config
#

site_cluster_ignition=$( merge_ignition_config "${CLUSTER_IGNITION}" "${DISCOVERY_IGN}" )
if [ -z "${site_cluster_ignition}" ]; then
    echo "Error processing cluster ignitionConfigOverride" >&2
    exit 1
fi

site_node_ignition=$( merge_ignition_config "${NODE_IGNITION}" "${BOOT_IGN}" )
if [ -z "${site_node_ignition}" ]; then
    echo "Error processing node ignitionConfigOverride" >&2
    exit 1
fi

site_node_installer_args=$(update_installer_args)
if [ -z "${site_node_installer_args}" ]; then
    echo "Error processing node installerArgs" >&2
    exit 1
fi

export site_cluster_ignition
export site_node_ignition
export site_node_installer_args

yq --inplace '
    with( '"${CLUSTER_IGNITION}"' = strenv(site_cluster_ignition) ; . )
    | with( '"${NODE_IGNITION}"' = strenv(site_node_ignition) ; . )
    | with( '"${NODE_INSTALLER_ARGS}"' = strenv(site_node_installer_args) ; . )
    ' "${SITECFG}"

