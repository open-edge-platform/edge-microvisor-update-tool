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
GetCommand "dd"
dd=$FNRETURN
GetCommand "fdisk"
fdisk=$FNRETURN
GetCommand "losetup"
losetup=$FNRETURN
GetCommand "sync"
sync=$FNRETURN
GetCommand "mkdir"
mkdir=$FNRETURN
GetCommand "mount"
mount=$FNRETURN
GetCommand "umount"
umount=$FNRETURN
GetCommand "rm"
rm=$FNRETURN
GetCommand "cp"
cp=$FNRETURN
GetCommand "find"
find=$FNRETURN
GetCommand "head"
head=$FNRETURN
GetCommand "findmnt"
findmnt=$FNRETURN
GetCommand "grep"
grep=$FNRETURN
GetCommand "lsblk"
lsblk=$FNRETURN
GetCommand "basename"
basename=$FNRETURN
GetCommand "chroot"
chroot=$FNRETURN
##

function verify_img {
    output=$($fdisk -l "$RAWSOURCE" 2>&1)
    check_error "Error encountered while running fdisk on $RAWSOURCE: $output"

    log_debug "fdisk output for $RAWSOURCE: $output"

    return 0
}

function setup_img {
    verify_img
    # Mount the raw image to loop
    local loop_device
    loop_device=$($losetup --find --show -P "$RAWSOURCE")
    check_error "Fail to setup loop device."
    update_config LOOPDEV "$loop_device"
    update_config SOURCEDEV "$loop_device""p2"
    update_config SOURCEBOOTDEV "$loop_device""p1"
    return 0
}

function verify_rootfs_loop {
    if check_device_exist "$SOURCEDEV"; then
        log_debug "Device $SOURCEDEV exists."
        return 0
    else
        log_error "Device $SOURCEDEV does not exist."
        return 1
    fi
}

function verify_boot_loop {
    if check_device_exist "$SOURCEBOOTDEV"; then
        log_debug "Device $SOURCEBOOTDEV exists."
        return 0
    else
        log_error "Device $SOURCEBOOTDEV does not exist."
        return 1
    fi
}

function write_rootfs {
    setup_img
    verify_rootfs_loop

    # Use dd to clone / directory of new raw image to second OS partition
    log_debug "flash image command"
    $dd if="$SOURCEDEV" of="$TARGETDEV" bs=4M

    check_error "Failed: Flashing Latest OS"
    $sync

    log_debug "Success: Flashing Latest OS."
}

function copy_rootfs {
    setup_img
    verify_rootfs_loop

    # mount update dir
    create_secure_dir "$SOURCECOPY"
    create_secure_dir "$DESTCOPY"
    $mount -o ro "$SOURCEDEV" "$SOURCECOPY"
    $mount "$TARGETDEV" "$DESTCOPY"
    # delete old rootfs and copy new
    $rm -rf "${DESTCOPY:?}"/*
    check_error "Failed: Remove old OS"
    $sync
    $cp -rp "$SOURCECOPY"/* "$DESTCOPY"/
    check_error "Failed: Copy Latest OS"
    $sync
    # unmount update dir
    $umount "$SOURCECOPY"
    $umount "$DESTCOPY"

    $rm -r "$SOURCECOPY"
    $rm -r "$DESTCOPY"

    log_debug "Success: Flashing Latest OS."
}

function copy_boot {
    verify_boot_loop

    if [ ! -d "$SOURCEBOOTMOUNT" ]; then
        $mkdir -p "$SOURCEBOOTMOUNT"
        log_debug "Folder '$SOURCEBOOTMOUNT' created."
    else
        log_debug "Folder '$SOURCEBOOTMOUNT' already exists."
    fi

    $mount -o ro "$SOURCEBOOTDEV" "$SOURCEBOOTMOUNT"
    local efi_source
    efi_source=$($find "$SOURCEBOOTMOUNT$SOURCEEFIMOUNT" -maxdepth 1 -type f -print | $head -n 1)

    log_debug "Copy UKI command"
    $dd if="$efi_source" of="$BOOTDATA"/linux.bak bs=4M

    check_error "Failed: Copy UKI"
    $sync

    $umount "$SOURCEBOOTDEV"
    wait_for_condition "! $mount | $grep -q $SOURCEBOOTDEV"
    $rm -r "$SOURCEBOOTMOUNT"

    log_debug "Success: Copy UKI."
}

function apply_boot {

    local active_UKI="$1"
    local next_uki=""

    log_debug "active UKI: $active_UKI"

    if [[ "$active_UKI" =~ -2\.efi$ ]]; then
        next_uki=$EFILOCATION/linux.efi
    else
        next_uki=$EFILOCATION/linux-2.efi
    fi

    log_debug "Apply UKI command"
    $dd if="$BOOTDATA"/linux.bak of="$next_uki" bs=4M

    check_error "Failed: Apply UKI"
    $sync

    log_debug "Success: Apply UKI."
}

function create_active_uki {
    local first_file
    first_file=$($find "$EFILOCATION" -maxdepth 1 -type f -print | $head -n 1)

    log_debug "Rename first UKI command"
    $dd if="$first_file" of="$EFILOCATION"/linux.efi bs=4M

    check_error "Failed: Rename first UKI."
    $sync
    $rm "$first_file"
}

function create_temp {
    delete_data
    create_secure_dir "$BOOTDATA"
    create_secure_temp_dir "$TEMPDATA"
    update_config TEMPDATA "$FNRETURN"
    resourced_tempdata_vars
}

function delete_data {
    $rm -rf "$BOOTDATA"
    $rm -rf "$TEMPDATA"
}

function remove_loop {

    # get loop device
    local loop_device
    loop_device=$($losetup -j "$RAWSOURCE" | awk -F: '{print $1}')
    update_config LOOPDEV "$loop_device"

    if [ -z "$LOOPDEV" ]; then
        return 0
    fi

    if ! check_device_exist "$LOOPDEV"; then
        log_debug "Device $LOOPDEV already removed."
        return 0
    fi

    log_debug "Remove $LOOPDEV"

    check_and_unmount_partitions "$LOOPDEV"

    $losetup -d "$LOOPDEV"
    check_error "Failed: Clean up loop device"

    update_config LOOPDEV ""
    log_debug "Device $LOOPDEV Removed."
}

function check_and_unmount_partitions {
    local target_partition="$1"

    mounted_partitions=$($findmnt -nr -o SOURCE | $grep "^$target_partition")

    if [ -n "$mounted_partitions" ]; then   
        # Unmount each partition
        for partition in $mounted_partitions; do
            $umount "$partition" && log_debug "$partition unmounted successfully."
        done
    else
        log_debug "No $target_partition partitions are mounted."
    fi
}

function check_device_exist {
    device_name=$($basename "$1")

    if [ -z "$device_name" ]; then
        log_debug "Cant check empty device exists."
        return 0
    fi

    if $lsblk | $grep -q "$device_name"; then
        log_debug "$device_name is present in lsblk."
        return 0
    else
        log_debug "$device_name is not present in lsblk."
        return 1
    fi
}

function clean_temp {
    # remove temp
    exec 200>&-
    rm -f "$RAWSOURCE.lock"
    $rm -rf "$RAWSOURCE"
    $rm -rf "$TEMPDATA"
}

function relabel_selinux {
    create_secure_dir "$UPDATEMOUNTPOINT"
    $mount "$TARGETDEV" "$UPDATEMOUNTPOINT"
    $mount --bind /dev "$UPDATEMOUNTPOINT"/dev
    $mount --bind /dev/pts "$UPDATEMOUNTPOINT"/dev/pts
    $mount --bind /proc "$UPDATEMOUNTPOINT"/proc
    $mount --bind /sys "$UPDATEMOUNTPOINT"/sys

    $chroot "$UPDATEMOUNTPOINT" /bin/bash <<EOT
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /
EOT

    $umount "$UPDATEMOUNTPOINT"/sys
    $umount "$UPDATEMOUNTPOINT"/proc
    $umount "$UPDATEMOUNTPOINT"/dev/pts -l
    $umount "$UPDATEMOUNTPOINT"/dev -l
    $umount "$UPDATEMOUNTPOINT"/ -l

    # unmount update dir
    $umount "$UPDATEMOUNTPOINT"
    $rm -r "$UPDATEMOUNTPOINT"
}

function add_login {
    local chroot_dir="$UPDATEMOUNTPOINT"
    local username="user"

    if [ -z "$chroot_dir" ] || [ -z "$username" ]; then
        echo "Failed on add_login to add login user"
        return 1
    fi

    create_secure_dir "$chroot_dir"
    $mount "$TARGETDEV" "$chroot_dir"
    $mount --bind /dev "$chroot_dir"/dev
    $mount --bind /dev/pts "$chroot_dir"/dev/pts
    $mount --bind /proc "$chroot_dir"/proc
    $mount --bind /sys "$chroot_dir"/sys

    export username

    sudo chroot "$chroot_dir" /bin/bash <<EOF
    # Create the user
    useradd -m -s /bin/bash "$username"

    # Set the password, escape special chars like dollar sign using slash else will failed    
    echo '$username:\$6\$BTZupwxuptVcnJ2q\$aKz3z0XxjPW0EI7r90/xfgMH.2J5dNB9V2jPbFPu0.NwioQh66VmyjVrG2uQuJnUu2d3MSvHqUiqGdU0VxFKA0' | chpasswd -e
    
    # Add the user to the sudo group
    usermod -aG sudo "$username"
EOF

    # Capture the exit status of the previous chroot command
    chroot_exit_status=$?
    if [ $chroot_exit_status -eq 0 ]; then
        echo "User $username added successfully in chroot environment $chroot_dir."
    else
        echo "Failed to add user $username in chroot environment $chroot_dir."
        return 1
    fi

    $umount "$chroot_dir"/sys
    $umount "$chroot_dir"/proc
    $umount "$chroot_dir"/dev/pts -l
    $umount "$chroot_dir"/dev -l
    $umount "$chroot_dir"/ -l

    # unmount update dir
    $umount "$chroot_dir"
    $rm -r "$chroot_dir"
}

function copy_onboarding_var {
    # mount update dir
    create_secure_dir "$DESTCOPY"
    $mount "$TARGETDEV" "$DESTCOPY"
    # copy all onboarding var
    cp /etc/hostname "$DESTCOPY/etc"
    #
    # unmount update dir
    $umount "$DESTCOPY"
    $rm -r "$DESTCOPY"

    log_debug "Success: Copy onboarding var."
}
