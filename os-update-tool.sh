#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

set +xe

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    exit 1
fi

# Handle race conditions
lockfile="/var/lock/$(basename "$0").lock"
lfd=88
eval "exec $lfd<>$lockfile"
if (! flock -xn "$lfd" 2>/dev/null); then
    echo "This script is already running, abort!"
    exit 1
fi
trap 'flock -u "$lfd"' EXIT

# Flags for handling options
# Active partition version
flag_v=false
# apply
flag_a=false
# commit applied config
flag_c=false
# write image
flag_w=false
# fde on
flag_fde=false
#dm verity on
flag_dmverity=false
# dev mode
flag_dev=false

IMG_SOURCE=""
IMG_SOURCE_SHA=""
osabupdate="/usr/bin/os-ab-update"

# Function to display help
display_help() {
    echo "Usage: sudo os-update-tool.sh [-v] [-a] [-c] [-w] [-u string] [-s string] [-h] [--debug]"
    echo
    echo "Options:"
    echo "  -v      Display current active partition."
    echo "  -a      Apply updated image as next boot."
    echo "  -c      Commit Updated image as default boot."
    echo "  -w      Write rootfs partition."
    echo "  -u      Define update image source path."
    echo "  -s      Define checksum value for the update image."
    echo "  -h      Display this help message."
    echo "  --debug Executes with debug log."
    exit 1
}

# print current version 
echo "os-update-tool ver-$(cat "$SRC_DIR"/VERSION)"
echo ""

# no parameter
if [[ "$#" -eq 0 ]]; then
    echo "No parameters were passed."
    display_help
fi

# parse flag
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -v )
            flag_v=true
            ;;
        -a )
            flag_a=true
            ;;
        -c )
            flag_c=true
            ;;
        -w )
            flag_w=true
            ;;
        -u )
            args=("$@")
            # not exist or empty
            if [[ -z "${args[1]+x}" || -z "${args[1]}" ]]; then
                echo "-u requires string input"
                display_help
            fi
            usrinput="${args[1]}"
            # Remove trailing space
            usr_source=${usrinput%% }
             if [[ -z "$usr_source" ]]; then
                echo "-u requires a valid file system path."
                exit 1
            fi
            export IMG_SOURCE="$usr_source"
            shift
            ;;
        -h )
            display_help
            ;;
        --debug )
            export DEBUG="true"
            ;;
        --dev )
            flag_dev=true
            ;;
        -s )
            args=("$@")
            # not exist or empty
            if [[ -z "${args[1]+x}" || -z "${args[1]}" ]]; then
                echo "-u requires string input"
                display_help
            fi
            usrinput="${args[1]}"
            # Remove trailing space
            sha_value=${usrinput%% }
            if [[ ! "$sha_value" =~ ^[a-fA-F0-9]{64}$ ]]; then
                echo "Invalid SHA-256 hash for -s."
                exit 1
            fi
            export IMG_SOURCE_SHA="$sha_value"
            shift
            ;;        
        * )
            echo "Invalid option given: -$1" 1>&2
            display_help
            ;;
    esac
    shift
done

# check for version
if $flag_v; then
    $osabupdate display
    exit 0
fi

# write image
if $flag_w; then
    if $flag_dev; then
        $osabupdate write "$IMG_SOURCE" "$IMG_SOURCE_SHA" "--dev"
    else
        $osabupdate write "$IMG_SOURCE" "$IMG_SOURCE_SHA"
    fi
fi

# apply update
if $flag_a; then
    $osabupdate apply
fi

# commit update
if $flag_c; then
    $osabupdate commit
fi

# Exit the script
exit 0
