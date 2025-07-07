#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

set +xe

sourced_var="SOURCED_$(echo "${BASH_SOURCE[0]}" | md5sum | sed -r 's/\s+\S+$//')"
if [ "${!sourced_var}" = true ]; then
    return 0
else
    eval "${sourced_var}=true"
fi
if [ "${BASH_SOURCE[0]}" -ef "$0" ]; then
    echo "Hey, you should source this script, not execute it!"
    exit 1
fi

TIMESTAMP_ENABLE=${TIMESTAMP_ENABLE:-true}

SCRIPT_DIR=$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)

if [[ -f "$SCRIPT_DIR/os-update-tool.config" ]]; then
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/os-update-tool.config"
else
    echo "Configuration file not found: $SCRIPT_DIR/os-update-tool.config"
    exit 1
fi

function get_timestamp {
    local timestamp=

    if [ "$TIMESTAMP_ENABLE" == "true" ]; then
            timestamp=$(date -I'seconds')
    fi

    echo "$timestamp"
}

function log_error {
    echo -e "$(get_timestamp) \033[31m[ERROR]\033[0m: $*" >&2
    echo
}

function log_warn {
    echo -e "$(get_timestamp) \033[93m[WARN]\033[0m: $*" >&2
    echo
}


function log_info {
    echo "$(get_timestamp) [INFO]: $*"  >&2
    echo
}

function log_debug {
    if [ "$DEBUG" == "true" ]; then
        echo "$(get_timestamp) [DEBUG]: $*"
        echo
    fi
}
