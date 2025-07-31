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

# sanitize $0 to ensure no command injection
mydir=$(cd -- "$(dirname "$0" | sed 's/[^a-zA-Z0-9._/-]//g')" && pwd -P)
SRC_DIR="$mydir/os-update-modules"

# Source all utilities, ensuring that they are readable and executable shell scripts
if [ -d "$SRC_DIR" ]; then
    for file in "$SRC_DIR"/*; do
        # Only source files that are readable and regular shell scripts (.sh)
        if [ -r "$file" ] && [[ "$file" == *.sh ]]; then
            # shellcheck source=/dev/null
            . "$file"
        fi
    done
fi

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
    log_error "No parameters were passed."
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
                log_error "-u requires string input"
                display_help
            fi
            usrinput="${args[1]}"
            # Remove trailing space
            usr_source=${usrinput%% }
            if ! is_valid_path "$usr_source"; then
                log_error "-u requires a valid file system path."
                exit 1
            fi
            update_config IMG_SOURCE "$usr_source"
            shift
            ;;
        -h )
            display_help
            ;;
        --debug )
            update_config DEBUG "true"
            ;;
        --dev )
            flag_dev=true
            ;;
        -s )
            args=("$@")
            # not exist or empty
            if [[ -z "${args[1]+x}" || -z "${args[1]}" ]]; then
                log_error "-u requires string input"
                display_help
            fi
            usrinput="${args[1]}"
            # Remove trailing space
            sha_value=${usrinput%% }
            if [[ ! "$sha_value" =~ ^[a-fA-F0-9]{64}$ ]]; then
                log_error "Invalid SHA-256 hash for -s."
                exit 1
            fi
            update_config IMG_SOURCE_SHA "$sha_value"
            shift
            ;;        
        * )
            log_error "Invalid option given: -$1" 1>&2
            display_help
            ;;
    esac
    shift
done

# check dmverity on
if check_dmverity_on; then
    flag_dmverity=true
fi

# check fde on
if check_fde_on; then
    flag_fde=true
fi

# check for version
if $flag_v; then
    log_debug "Display active partition."

    # fde on
    if $flag_fde; then
        check_real_active_partition
    elif $flag_dmverity; then
        check_dm_active_partition
    else
        check_active_partition
    fi

    echo "Active partition: $FNRETURN"
    check_error "Failed to display partition."

    log_debug "Display active image ID."

    get_image_id
    echo "Image UUID: $FNRETURN"
    check_error "Failed to get image ID."

    exit 0
fi

# write image
if $flag_w; then
    log_info "Write Image."
    
    # fde on
    if $flag_fde; then
        get_real_inactive_partition
    else
        get_first_inactive_partition "$flag_dmverity"
    fi
    
    update_config TARGETDEV "$FNRETURN"

    verify_img_sha
    check_error "Failed check image sha." 3

    if [[ -z "$TARGETDEV" ]]; then
        log_error "No Suitable Partition for Update Found."
        exit 1
    fi

    # create temporary memory for os-update-tool
    create_temp

    download_img
    check_error "Failed image extraction." 3

    remove_loop

    # fde on
    if $flag_fde; then
        copy_rootfs
    else
        write_rootfs
    fi

    check_error "Failed write image." 3

    copy_onboarding_var

    check_error "Failed copy onboarding var." 3

    copy_boot

    check_error "Failed write boot." 3
   
    if $flag_dev; then
        log_debug "Dev mode enabled."
        # temporary add login
        add_login
        check_error "Failed add login user" 3
    fi
    
    relabel_selinux

    check_error "Failed relabel selinux." 3

    # fde on
    if $flag_fde; then
        get_boot_uuid
        log_debug "change luksUUID to $FNRETURN"
        set_luks_uuid "$TARGETDEV" "$FNRETURN"
        check_error "Failed Set luks UUID." 3
    else
        get_partition_uuid "$SOURCEDEV"
        log_debug "change partuuid to $FNRETURN"
        set_partition_uuid "$TARGETDEV" "$FNRETURN" "$flag_dmverity"
        check_error "Failed Set part UUID." 3
    fi

    if $flag_dmverity; then
        set_verity "$TARGETDEV" "$flag_fde"
        check_error "Failed Set Verity." 3
    fi

    remove_loop
    if $ISDELETESOURCE; then
        clean_temp
    fi

    log_info "Write image successfull."
fi

# apply update
if $flag_a; then
    log_info "Apply updates."

    get_previous_target
    update_config TARGETDEV "$FNRETURN"

    if [ -z "$TARGETDEV" ]; then
        log_error "Nothing to apply."
        exit 1
    fi

    if ! is_active_uki_exist; then
        create_active_uki
    fi

    # fde on
    if $flag_fde; then
        check_real_active_partition
    elif $flag_dmverity; then
        check_dm_active_partition
    else
        check_active_partition
    fi
    active_boot="$FNRETURN"

    get_active_uki "$active_boot" "$flag_fde" "$flag_dmverity"
    active_efi="$FNRETURN"

    log_debug "Active Partition: $active_boot"
    log_debug "Active UKI: $active_efi"

    apply_boot "$active_efi"

    create_boot_config "$active_efi"

    set_default_boot

    set_next_boot

    check_error "Failed apply boot changes." 3

    log_info "Apply update successfull."
fi

# commit update
if $flag_c; then
    log_info "Commit update."

    # check if updates has been done before
    if ! is_valid_path "$BOOTDATA"/linux.bak; then
        log_error "Nothing to commit."
        exit 1
    fi

    # fde on
    if $flag_fde; then
        check_real_active_partition
    elif $flag_dmverity; then
        check_dm_active_partition
    else
        check_active_partition
    fi
    active_boot="$FNRETURN"

    get_active_uki "$active_boot" "$flag_fde" "$flag_dmverity"
    check_error "Failed to get active UKI." 4
    active_efi="$FNRETURN"

    log_debug "Active Partition: $active_boot"
    log_debug "Active UKI: $active_efi"

    commit "$active_efi"
    check_error "Failed to change boot." 4
    
    delete_data
    check_error "Failed to commit update OS." 4

    log_info "Commit update successfull."
fi

# Exit the script
exit 0
