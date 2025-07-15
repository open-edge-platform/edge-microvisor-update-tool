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
GetCommand "bootctl"
bootctl=$FNRETURN
GetCommand "blkid"
blkid=$FNRETURN
GetCommand "basename"
basename=$FNRETURN
GetCommand "grep"
grep=$FNRETURN
GetCommand "uniq"
uniq=$FNRETURN
GetCommand "head"
head=$FNRETURN
GetCommand "cryptsetup"
cryptsetup=$FNRETURN
GetCommand "lsblk"
lsblk=$FNRETURN
GetCommand "cut"
cut=$FNRETURN
GetCommand "tune2fs"
tune2fs=$FNRETURN
##

function create_boot_config {
    # usage create_boot_config linux.efi
    local active_UKI="$1"
    local next_boot=""
    local active_boot=""

    # active uki has -2
    if [[ "$active_UKI" =~ -2\.efi$ ]]; then
        active_boot="linux-2.efi"
        next_boot="linux.efi"
    else
        active_boot="linux.efi"
        next_boot="linux-2.efi"
    fi

    update_config NEXTBOOT $next_boot
    update_config ACTIVEBOOT $active_boot

    log_debug "Boot config setup completed successfully."
}

function set_default_boot {
    $bootctl set-default "$ACTIVEBOOT"
    check_error "Fail to set default OS"

    log_debug "Set default boot successfully."

    return 0
}

function set_next_boot {
    $bootctl set-oneshot "$NEXTBOOT"
    check_error "Fail to set next boot OS"

    log_debug "Set next boot successfully."

    return 0
}

function commit {
    local active_UKI="$1"
    local cur_UKI=""
    # active uki has -2
    if [[ "$active_UKI" =~ -2\.efi$ ]]; then
        cur_UKI="linux-2.efi"
    else
        cur_UKI="linux.efi"
    fi

    log_debug "Next Default $cur_UKI."

    $bootctl set-default "$cur_UKI"

    check_error "Fail to set default OS"

    log_debug "Set default boot successfully."
}

function restore {
    local active_UKI="$1"
    local prev_uki=""
    log_debug "active UKI: $active_UKI"
    # active uki has -2
    if [[ "$active_UKI" =~ -2\.efi$ ]]; then
        prev_uki="linux.efi"
    else
        prev_uki="linux-2.efi"
    fi

    log_debug "Next Default $prev_uki."

    $bootctl set-default "$prev_uki"

    check_error "Fail to restore previous OS"

    log_debug "Restore boot successfully."
}

function is_uki_partition_match {
    # usage is_uki_partition_match /dev/sad4 linux.efi
    # Input arguments: partition (e.g., /dev/sda4) and UKI file name
    local partition="$1"
    local uki_filename
    uki_filename=$($basename "$2")

    update_config FNRETURN ""

    # Get the PARTUUID of the given partition
    local partuuid_partition
    partuuid_partition=$($blkid -s PARTUUID -o value "$partition")

    if [ -z "$partuuid_partition" ]; then
        log_error "Error: Could not retrieve PARTUUID for $partition"
        return 1
    fi

    # Extract the PARTUUID from bootctl list based on the UKI file name
    local partuuid_uki
    partuuid_uki=$($bootctl list | $grep -A 5 "$uki_filename" | $grep -oP 'root=PARTUUID=\K[^ ]+' | $uniq | $head -n 1)

    if [ -z "$partuuid_uki" ]; then
        partuuid_uki=$( $bootctl list | $grep -A 5 "$uki_filename" | $grep -oP 'boot_uuid=\K[^ ]+' | $uniq | $head -n 1 )
        if [ -z "$partuuid_uki" ]; then
            log_error "Error: Could not retrieve PARTUUID for UKI file $uki_filename in bootctl"
            return 1
        fi
    fi

    # Compare the two PARTUUIDs
    if [ "$partuuid_partition" = "$partuuid_uki" ]; then
        update_config FNRETURN "yes"
    else
        update_config FNRETURN "no"
    fi
}

function is_uki_luks_match {
    # usage is_uki_luks_match /dev/sad4 linux.efi
    # Input arguments: partition (e.g., /dev/sda4) and UKI file name
    local partition="$1"
    local uki_filename
    uki_filename=$($basename "$2")

    update_config FNRETURN ""

    # Get the luks UUID of the given partition
    local luksuuid_partition
    luksuuid_partition=$($cryptsetup luksUUID "$partition")

    if [ -z "$luksuuid_partition" ]; then
        log_error "Error: Could not retrieve luks UUID for $partition"
        return 1
    fi

    # Extract the boot UUID from bootctl list based on the UKI file name
    local luksuuid_uki
    luksuuid_uki=$($bootctl list | $grep -A 5 "$uki_filename" | $grep -oP 'boot_uuid=\K[^ ]+' | $uniq | $head -n 1)

    if [ -z "$luksuuid_uki" ]; then
        log_error "Error: Could not retrieve luks UUID for UKI file $uki_filename in bootctl"
        return 1
    fi

    # Compare the two PARTUUIDs
    if [ "$luksuuid_partition" = "$luksuuid_uki" ]; then
        update_config FNRETURN "yes"
    else
        update_config FNRETURN "no"
    fi
}

function is_uki_uuid_match {
    # usage is_uki_uuid_match /dev/sad4 linux.efi
    # Input arguments: partition (e.g., /dev/sda4) and UKI file name
    local partition="$1"
    local uki_filename
    uki_filename=$($basename "$2")

    update_config FNRETURN ""

    # Get the UUID of the given partition
    local uuid_partition
    uuid_partition=$($tune2fs -l "$partition" | $grep 'Filesystem UUID' | awk '{print $3}')

    if [ -z "$uuid_partition" ]; then
        log_error "Error: Could not retrieve UUID for $partition"
        return 1
    fi

    # Extract the boot UUID from bootctl list based on the UKI file name
    local uuid_uki
    uuid_uki=$($bootctl list | $grep -A 5 "$uki_filename" | $grep -oP 'boot_uuid=\K[^ ]+' | $uniq | $head -n 1)

    if [ -z "$uuid_uki" ]; then
        log_error "Error: Could not retrieve UUID for UKI file $uki_filename in bootctl"
        return 1
    fi

    # Compare the two PARTUUIDs
    if [ "$uuid_partition" = "$uuid_uki" ]; then
        update_config FNRETURN "yes"
    else
        update_config FNRETURN "no"
    fi
}

function is_active_uki_exist {
    local path=$EFILOCATION"/linux.efi"
    if [ -f "$path" ]; then
        return 0
    else
        return 1
    fi
}

function get_active_uki {
    local active_partition="$1"
    local fde="$2"
    local dmv="$3"
    
    update_config FNRETURN ""
    # Loop through all .efi files in the directory
    for file in "$EFILOCATION"/*.efi; do
        if [[ -f "$file" ]]; then
            if $fde; then
                is_uki_luks_match "$active_partition" "$file"
            elif $dmv; then 
                is_uki_uuid_match "$active_partition" "$file"
            else
                is_uki_partition_match "$active_partition" "$file"
            fi
            if [[ "$FNRETURN" == "yes" ]]; then
                update_config FNRETURN "$file"
                return 0
            fi
        fi
    done
    return 1
}

function get_previous_target {
    update_config FNRETURN ""

    # get boot uuid
    boot_uuid=$( $grep -a -h -o "boot_uuid=.* " "$BOOTDATA"/linux.bak | $cut -c 11-46 | $head -1)

    if [ -z "$boot_uuid" ]; then
        boot_uuid=$( $grep -a -h -o "PARTUUID=.* " "$BOOTDATA"/linux.bak | $cut -c 10-45 | $head -1)
    fi

    log_debug "Found boot_uuid=$boot_uuid."

    # Use lsblk to list partitions and filter for type "part"
    local partitions
    IFS=$'\n' read -d '' -r -a partitions < <($lsblk -no NAME,TYPE | awk '$2 == "part" {print $1}')

    # Loop through each partition
    for partition in "${partitions[@]}"; do
        cleanName "$clean_path"
        clean_path=$FNRETURN

        update_config FNRETURN ""

        log_debug "Check partition: /dev/$clean_path"

        rootfs_match=$($blkid | $grep -i "$boot_uuid" | $grep -i "/dev/$clean_path")

        if [[ -n "$rootfs_match" ]]; then
            update_config FNRETURN "/dev/$clean_path"
            return 0
        fi
    done
}
