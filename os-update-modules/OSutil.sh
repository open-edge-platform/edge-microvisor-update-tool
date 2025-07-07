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
GetCommand "lsblk"
lsblk=$FNRETURN
GetCommand "grep"
grep=$FNRETURN
GetCommand "cut"
cut=$FNRETURN
GetCommand "head"
head=$FNRETURN
GetCommand "blkid"
blkid=$FNRETURN
GetCommand "find"
find=$FNRETURN
GetCommand "sfdisk"
sfdisk=$FNRETURN
GetCommand "basename"
basename=$FNRETURN
GetCommand "cryptsetup"
cryptsetup=$FNRETURN
GetCommand "veritysetup"
veritysetup=$FNRETURN
GetCommand "mkdir"
mkdir=$FNRETURN
GetCommand "mount"
mount=$FNRETURN
GetCommand "umount"
umount=$FNRETURN
GetCommand "rm"
rm=$FNRETURN
GetCommand "dmsetup"
dmsetup=$FNRETURN
GetCommand "sync"
sync=$FNRETURN
GetCommand "tune2fs"
tune2fs=$FNRETURN
GetCommand "e2fsck"
e2fsck=$FNRETURN
##

function check_active_partition {
    log_debug "Get Active Partition."
    update_config FNRETURN ""

    local current_name=""

    while IFS= read -r line; do
        read -r name fstype mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN

        update_config FNRETURN ""

        if [ "$mountpoint" = "/" ]; then
            if [[ "$name" == *"verity"* ]]; then
                update_config FNRETURN "/dev/$current_name"
                log_debug "Active partition: /dev/$current_name"
            else
                update_config FNRETURN "/dev/$name"
                log_debug "Active partition: /dev/$name"
            fi
            break
        fi

        if [[ -n "$name" ]]; then
            current_name="$name"
        fi

    done < <($lsblk -nr -o NAME,FSTYPE,MOUNTPOINT)

    return 0
}

function check_real_active_partition {
    log_debug "Get Real Active Partition."
    update_config FNRETURN ""

    local current_name=""
    local current_subname=""

    while IFS= read -r line; do
        read -r name subname _ mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN
        cleanName "$subname"
        subname=$FNRETURN

        update_config FNRETURN ""

        # Check if the mountpoint is "/"
        if [[ "$mountpoint" == "/" && "$current_subname" == rootfs* ]]; then
            update_config FNRETURN "/dev/$current_name"
            log_debug "Active partition: /dev/$current_name"
            break
        fi

        if [[ -n "$name" ]]; then
            current_name="$name"
        fi

        if [[ -n "$subname" ]]; then
            current_subname="$subname"
        fi

    done < <($lsblk -n -o NAME,FSTYPE,MOUNTPOINT)

    return 0
}

function check_dm_active_partition {
    log_debug "Get DM Verity Active Partition."
    update_config FNRETURN ""

    local current_name=""
    local current_subname=""

    while IFS= read -r line; do
        read -r name _ _ mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN

        update_config FNRETURN ""

        # Check if the mountpoint is "/"
        if [[ "$mountpoint" == "/" ]]; then
            fsindex=$( echo "/dev/$current_name" | awk -F ":" '{print substr($1,length($1),1)}' )
            if [[ "$fsindex" == "$INDEXPARTA" || "$fsindex" == "$INDEXPARTB" ]]; then
                update_config FNRETURN "/dev/$current_name"
                log_debug "Active partition: /dev/$current_name"
                break
            fi
        fi

        if [[ -n "$name" ]]; then
            current_name="$name"
        fi

    done < <($lsblk -n -o NAME,FSTYPE,MOUNTPOINT)

    return 0
}

function get_first_inactive_partition {
    log_debug "Get inactive partition."
    update_config FNRETURN ""

    local dmv="$1"

    # get active partition
    if $dmv; then
        check_dm_active_partition
    else
        check_active_partition
    fi

    active_part=$FNRETURN

    update_config FNRETURN ""

    while IFS= read -r line; do
        read -r name fstype partlabel mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN

        update_config FNRETURN ""

        if [ -z "$mountpoint" ] && [ -n "$fstype" ] && [[ "$partlabel" == rootfs* ]] && ! [[ "$name" == loop* ]] && ! [[ "/dev/$name" == "$active_part" ]]; then
            fsindex=$( echo "/dev/$name" | awk -F ":" '{print substr($1,length($1),1)}' )
            if [[ "$fsindex" != "$INDEXPARTA" && "$fsindex" != "$INDEXPARTB" ]]; then
                log_error "Partition index must be either $INDEXPARTA or $INDEXPARTB. Found: $fsindex"
                return 1
            fi
            update_config FNRETURN "/dev/$name"
            log_debug "Inactive Partiton : /dev/$name"
            break
        fi
    done < <($lsblk -nr -o NAME,FSTYPE,PARTLABEL,MOUNTPOINT)

    return 0
}

function get_real_inactive_partition {
    log_debug "Get inactive partition."
    update_config FNRETURN ""

    # get active first
    local active_name=""
    local active_subname=""

    while IFS= read -r line; do
        read -r name subname _ mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN
        cleanName "$subname"
        subname=$FNRETURN

        update_config FNRETURN ""

        # Check if the mountpoint is "/"
        if [[ "$mountpoint" == "/" ]]; then
            # assume the other is not active
            if [[ "$active_name" == rootfs_a* || "$active_subname" == rootfs_a* ]]; then
                update_config FNRETURN "$CRYPTEDPART/rootfs_b"
                log_debug "Inactive Partiton : $CRYPTEDPART/rootfs_b"
            else
                update_config FNRETURN "$CRYPTEDPART/rootfs_a"
                log_debug "Inactive Partiton : $CRYPTEDPART/rootfs_a"
            fi
            break
        fi

        # Accumulate values for name and subname
        if [[ -n "$name" ]]; then
            active_name="$name"
        fi
        if [[ -n "$subname" ]]; then
            active_subname="$subname"
        fi
    done < <($lsblk -n -o NAME,FSTYPE,MOUNTPOINT)

    return 0
}

function get_image_id {
    log_debug "Get image ID from /etc/image-id."
    update_config FNRETURN ""
    # Extract the IMAGE_UUID from the file
    local image_uuid
    image_uuid=$($grep "^IMAGE_UUID=" /etc/image-id | $cut -d'=' -f2)
    log_debug "Image ID : $image_uuid"

    update_config FNRETURN "$image_uuid"

    return 0
}

function get_partition_uuid {
    # usage get_partition_uuid /dev/sda3
    local target_partition="$1"

    log_debug "Get Partition UUID : $target_partition"
    update_config FNRETURN ""

    local PARTUUID=""
    PARTUUID=$( $grep -a -h -o "boot_uuid=.* " "$BOOTDATA"/linux.bak | $cut -c 11-46 | $head -1 )

    if [[ -z "$PARTUUID" ]]; then
        log_debug "Boot_uuid not found in UKI will use part UUID."
        PARTUUID=$($blkid -s PARTUUID -o value "$target_partition")
    fi

    update_config FNRETURN "$PARTUUID"
}

function get_boot_uuid {
    local ukifile
    update_config FNRETURN ""
    ukifile=$($find "$EFILOCATION" -type f -name "*.bak" | $head -n 1)
    local rootfsUUID
    rootfsUUID=$( $grep -a -h -o "boot_uuid=.* " "$ukifile" | $cut -c 11-46 | $head -1)
    check_error "Failed get boot UUID."
    update_config FNRETURN "$rootfsUUID"
}

function set_partition_uuid {
    # usage set_partition_uuid /dev/sda3 NEW-UUID true
    local device_partition="$1"
    local new_uuid="$2"
    local dmv="$3"

    # check for multiple UUID in system
    rootfs_match=$($blkid | $grep -i "$new_uuid" | $grep -o '^[^:]*')

    if [[ -n "$rootfs_match" ]] && [[ "$rootfs_match" == "$device_partition" ]]; then
        log_debug "PARTUUID is the same."
        return 0
    fi

    if [[ -n "$rootfs_match" ]] && ! [[ "$rootfs_match" == *loop* ]]; then
        log_error "Duplicate PARTUUID detected please check image source."
        return 1
    fi

    if $dmv; then
        log_debug "change UUID"
        $e2fsck -f "$device_partition" -y
        $tune2fs -U "$new_uuid" "$device_partition"
        check_error "Failed to set UUID."
    else
        # Extract the base device (e.g., /dev/sda from /dev/sda3)
        local base_device
        base_device="${device_partition%[0-9]*}"

        # Extract the partition number (e.g., 3 from /dev/sda3)
        local partition_number
        partition_number=$(echo "$device_partition" | $grep -o '[0-9]*$')

        # Run the sfdisk command with the base device, partition number, and new UUID
        log_debug "change UUID"
        $sfdisk --part-uuid "$base_device" "$partition_number" "$new_uuid"
        check_error "Failed to set UUID."
    fi
}

function get_block_name {
    # extract rootfs name
    local device
    device=$($basename "$1")
    update_config FNRETURN ""

    # get device block
    local block_name=""

    log_debug "Get block name for $device."

    while IFS= read -r line; do     
        read -r name subname _ mountpoint <<< "$line"
        cleanName "$name"
        name=$FNRETURN
        cleanName "$subname"
        subname=$FNRETURN

        update_config FNRETURN ""

        if [[ "$subname" == "$device" ]]; then
            update_config FNRETURN "/dev/$block_name"
            break
        fi

        # Accumulate values for block
        if [[ -n "$name" ]]; then
            block_name="$name"
        fi

    done < <($lsblk -n -o NAME,FSTYPE,MOUNTPOINT)
}

function set_luks_uuid {
    # usage set_luks_uuid /dev/mapper/rootfs_a NEW-UUID
    local device_partition="$1"
    local new_uuid="$2"

    get_block_name "$device_partition"
    target_block="$FNRETURN"

    # check for multiple UUID in system
    rootfs_match=$($blkid | $grep -i "$new_uuid" | $grep -o '^[^:]*')

    if [[ -n "$rootfs_match" ]] && [[ "$rootfs_match" == "$device_partition" ]]; then
        log_debug "luksUUID is the same."
        return 0
    fi

    if [[ -n "$rootfs_match" ]] && ! [[ "$rootfs_match" == *loop* ]]; then
        log_error "Duplicate luksUUID detected please check image source."
        return 1
    fi

    log_debug "Change luksUUID for $target_block"
    $cryptsetup luksUUID --batch-mode "$target_block" --uuid "$new_uuid"
    check_error "Failed set luks UUID."
    
}

# Function to replace the number at the end of the device string
function replace_device_index {
    update_config FNRETURN ""
    local device=$1
    local new_index=$2
    update_config FNRETURN "${device%[0-9]*}$new_index"
}

function set_verity {
    local device_partition="$1"
    local fde="$2"
    # extract rootfs name
    local device

    log_debug "Set verity for $device_partition."

    $mkdir -p /tmp/temp
    if $fde; then
        log_debug "FDE is enabled."
        device=$($basename "$device_partition")
        $mount /dev/mapper/ver_roothash /tmp/temp
        if [[ "$device" == rootfs_b ]]; then
            log_debug "Verity Setup on rootfs_b"
            $veritysetup format /dev/mapper/rootfs_b /dev/mapper/root_b_ver_hash_map | $grep Root | $cut -f2 > /tmp/temp/part_b_roothash
        else
            log_debug "Verity Setup on rootfs_a"
            $veritysetup format /dev/mapper/rootfs_a /dev/mapper/root_a_ver_hash_map | $grep Root | $cut -f2 > /tmp/temp/part_a_roothash
        fi
    else
        log_debug "DM Verity is enabled."
        replace_device_index "$device_partition" "$INDEXHASHPART"
        $mount "$FNRETURN" /tmp/temp
        fsindex=$( echo "$device_partition" | awk -F ":" '{print substr($1,length($1),1)}' )
        if [[ "$fsindex" == "$INDEXPARTA" ]]; then
            log_debug "Verity Setup on $device_partition"
            replace_device_index "$device_partition" "$INDEXMAPPARTA"
            $veritysetup format "$device_partition" "$FNRETURN" | $grep Root | $cut -f2 > /tmp/temp/part_a_roothash
        else
            log_debug "Verity Setup on $device_partition"
            replace_device_index "$device_partition" "$INDEXMAPPARTB"
            $veritysetup format "$device_partition" "$FNRETURN" | $grep Root | $cut -f2 > /tmp/temp/part_b_roothash
        fi
    fi

    $sync

    $umount /tmp/temp && $rm -rf /tmp/temp
}

# Function to check if fde is enabled
function check_fde_on {
    # Execute the command and capture the output
    output=$($dmsetup ls --target crypt 2>&1)

    # Check if the output contains "No devices found"
    if [[ "$output" == *"No devices found"* ]]; then
        return 1  # Return false if "No devices found" is in the output
    fi

    # Check if the output is empty
    if [[ -z "$output" ]]; then
        return 1  # Return false if the output is empty
    else
        if [[ "$output" == *"rootfs_a"* && "$output" == *"rootfs_b"* ]]; then
            return 0  # found rootfs_a and rootfs_b
        fi
        return 1  # Return false if device not recognize
    fi
}

# Function to check if dmverity is enabled
function check_dmverity_on {
    # check double rootfs mounting "/"
    local root_count=0

    while IFS= read -r line; do
        read -r _ _ mountpoint <<< "$line"
        if [[ "$mountpoint" == "/" ]]; then
            ((root_count++))
        fi
    done < <($lsblk -nr -o NAME,FSTYPE,MOUNTPOINT)

    if [[ "$root_count" -ge 2 ]]; then
        return 0  # Return true if there are 2 or more "/" mountpoints
    else
        return 1  # Return false if there is less than 2 "/" mountpoints
    fi
}
