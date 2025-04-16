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

SCRIPT_DIR=$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)

if [[ -f "$SCRIPT_DIR/os-update-tool.config" ]]; then
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/os-update-tool.config"
else
    echo "Configuration file not found: $SCRIPT_DIR/os-update-tool.config"
    exit 1
fi

if [[ -f "$SCRIPT_DIR/log.sh" ]]; then
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/log.sh"
else
    echo "Configuration file not found: $SCRIPT_DIR/log.sh"
    exit 1
fi

if [[ -f "$SCRIPT_DIR/common.sh" ]]; then
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/common.sh"
else
    echo "Configuration file not found: $SCRIPT_DIR/common.sh"
    exit 1
fi

# get all used command
GetCommand "xz"
xz=$FNRETURN
GetCommand "gzip"
gzip=$FNRETURN
GetCommand "flock"
flock=$FNRETURN
GetCommand "sha256sum"
sha256sum=$FNRETURN
##

function download_img {
    if is_valid_path "$IMG_SOURCE"; then
        log_debug "$IMG_SOURCE is a file system path."
        update_config TEMPFILE "$IMG_SOURCE"
    else
        log_error "$IMG_SOURCE is not a valid file system path."
        return 1
    fi

    extract_img
    return 0
}

function extract_img {
    # Lock the file exclusively
    exec 200>"$RAWSOURCE.lock"
    $flock -n 200 || { log_error "Failed to acquire lock"; return 1; }
    
    if [[ "$TEMPFILE" == *.xz ]]; then
        check_file_creation "$RAWSOURCE" "$EXPECTEDRAWSIZE"
        CMD=("$xz" -dc "$TEMPFILE")
        "${CMD[@]}" > "$RAWSOURCE"

        check_error "Failed to extract .xz image."

        if [[ "$RAWSOURCE" != *.raw ]]; then
            log_error "Invalid File Type Detected."
            return 1
        fi    
    elif [[ "$TEMPFILE" == *.gz ]]; then
        check_file_creation "$RAWSOURCE" "$EXPECTEDRAWSIZE"
        CMD=("$gzip" -dc "$TEMPFILE")
        "${CMD[@]}" > "$RAWSOURCE"
        check_error "Failed to extract .gz image."
        if [[ "$RAWSOURCE" != *.raw ]]; then
            log_error "Invalid File Type Detected."
            return 1
        fi  
    else
        if [[ "$TEMPFILE" == *.raw ]]; then
            update_config RAWSOURCE "$TEMPFILE"
            update_config ISDELETESOURCE "false"
        else
            log_error "Invalid File Type Detected."
            return 1
        fi
    fi
}

function verify_img_sha {
    log_info "Verifying sha256 for raw image: $IMG_SOURCE $IMG_SOURCE_SHA"
    # Check if IMG_SOURCE and IMG_SOURCE_SHA are set
    if [[ -z "$IMG_SOURCE" || -z "$IMG_SOURCE_SHA" ]]; then
        log_error "Missing input: IMG_SOURCE or IMG_SOURCE_SHA is not set."
        return 1
    fi

    # Check if IMG_SOURCE exists
    if [[ ! -f "$IMG_SOURCE" ]]; then
        log_error "File not found: $IMG_SOURCE"
        return 1
    fi

    # Calculate the SHA-256 checksum of the IMG_SOURCE
    local calculated_sha
    calculated_sha=$($sha256sum "$IMG_SOURCE" | awk '{print $1}')

    # Compare the calculated SHA with the expected SHA
    if [[ "$calculated_sha" == "$IMG_SOURCE_SHA" ]]; then
        log_info "SHA256 verification successful for $IMG_SOURCE"
        return 0
    else
        log_error "SHA256 mismatch! Expected: $IMG_SOURCE_SHA, Got: $calculated_sha"
        return 1
    fi
}