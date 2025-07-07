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

# update a variable in the config file
function update_config {
    local config_file="$SCRIPT_DIR/os-update-tool.config"
    local key="$1"
    local new_value="$2"

    # Validate key: only allow alphanumeric characters and underscores
    if [[ ! "$key" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
        log_error "Invalid configuration key: $key. Keys must be alphanumeric and start with a letter or underscore."
        exit 1
    fi

    # Validate new_value: disallow dangerous characters but allow it to be empty
    if [[ -n "$new_value" && "$new_value" =~ [^a-zA-Z0-9._:/\ -] ]]; then
        log_error "Invalid value for $key: $new_value. Values must not contain special or dangerous characters."
        exit 1
    fi

    # Check if the key exists in the config file
    if grep -q "^$key=" "$config_file"; then
        # prevent writing back to file but still updates value
        export "$key"="$new_value"
        log_debug "Updated $key to $new_value"
    else
        log_error "$key is not a valid configuration."
        exit 1
    fi
}

# execute clean command
function GetCommand {
    local cmd="$1"
    log_debug "Cleaning Command: $cmd"
    update_config FNRETURN ""

    # Use `which` to get the full path of the command
    local cmd_path
    cmd_path=$(which "$cmd" 2>/dev/null)

    if [[ -z "$cmd_path" ]]; then
        log_error "Command '$cmd' not found"
        exit 1
    fi

    # Check for symlink
    local stat_output
    stat_output=$(stat -c '%N' "$cmd_path" 2>/dev/null)
    if [[ "$stat_output" == *'->'* ]]; then
        log_debug "Command '$cmd' is a symlink: $stat_output"
        # Resolve the symlink to the actual binary path
        cmd_path=$(readlink -f "$cmd_path" 2>/dev/null || echo "$cmd_path")
    fi

    # Get binary birthdate
    local cmd_birthdate
    cmd_birthdate=$(stat -c '%W' "$cmd_path")

    if [[ "$cmd_birthdate" -lt 0 ]]; then
        log_error "Unable to retrieve birthdate for $cmd_path"
        exit 1
    fi

    # Get /etc/os-release birthdate
    local os_birthdate
    os_birthdate=$(stat -c '%W' /etc/os-release)

    if [[ "$os_birthdate" -lt 0 ]]; then
        log_error "Unable to retrieve birthdate for /etc/os-release"
        exit 1
    fi

    # Convert epoch times to YYYY-MM-DD format for comparison
    local cmd_date os_date
    cmd_date=$(date -d "@$cmd_birthdate" +%Y-%m-%d)
    os_date=$(date -d "@$os_birthdate" +%Y-%m-%d)

    # Compare only the dates
    if [[ "$cmd_date" > "$os_date" ]]; then
        log_error "Command '$cmd' was created after the OS release file"
        exit 1
    fi

    log_debug "Resolved Path: $cmd_path"

    # Output the cleaned command path
    update_config FNRETURN "$cmd_path"
    return 0
}

function check_error {
    local status=$?
    local message=$1
    local override=${2:-$status}

    if [[ $status -ne 0 ]]; then
        log_error "$message"
        exit "$override"
    fi
}

# get all used command
GetCommand "dirname"
dirname=$FNRETURN
GetCommand "df"
df=$FNRETURN
GetCommand "tail"
tail=$FNRETURN
GetCommand "sleep"
sleep=$FNRETURN
GetCommand "bash"
bash=$FNRETURN
GetCommand "rm"
rm=$FNRETURN
GetCommand "mkdir"
mkdir=$FNRETURN
GetCommand "chmod"
chmod=$FNRETURN
GetCommand "mktemp"
mktemp=$FNRETURN
GetCommand "sed"
sed=$FNRETURN
##

function check_file_creation {
    local file_path=$1   # Full path to the file to be created
    local size=$2        # Desired file size in MB

    # Extract the directory from the full file path
    local dir
    dir=$($dirname "$file_path")

    # Get available space in the directory (in KB)
    available_space_kb=$($df --output=avail "$dir" | $tail -n 1)
    
    # Convert size from MB to KB for comparison
    required_space_kb=$((size * 1024))

    # Check if there is enough space
    if (( available_space_kb >= required_space_kb )); then
        log_debug "There is enough space to create a $size MB file at $file_path."
    else
        log_error "Not enough space to create a $size MB file at $file_path."
        exit 1
    fi
}

# check if a string is a valid file system path
is_valid_path() {
    local path=$1

    # Check if the path is not empty
    if [[ -z "$path" ]]; then
        log_error "Path is empty."
        return 1
    fi

    # Check if the path contains only allowed characters (alphanumeric, ".", "_", "-", "/")
    if [[ "$path" =~ ^[a-zA-Z0-9._/-]+$ ]]; then
        if [ -f "$path" ]; then
            return 0
        else
            log_error "Path does not exist"
            return 1
        fi
    else
        log_error "Path contains invalid characters."
        return 1
    fi
}

function wait_for_condition {
    local condition="$1"
    local interval=${2:-1}  # Optional: set interval between checks (default 1 seconds)
    local timeout=${3:-10}  # Optional: set timeout (default 10 seconds)
    local elapsed=0

    log_debug "Wait for $condition."

    while ! $bash -c "$condition"; do
        if (( elapsed >= timeout )); then
            log_error "Timeout waiting for $condition: $timeout seconds"
            return 1
        fi
        $sleep "$interval"
        (( elapsed += interval ))
    done

    return 0
}

function create_secure_dir {
    # Create a secure directory
    secure_dir="$1"
    
    # Remove existing directory if it exists (optional)
    $rm -rf "$secure_dir"
    
    # Create the directory with restrictive permissions
    $mkdir -p "$secure_dir"
    $chmod 700 "$secure_dir"
}

function create_secure_temp_dir {
    # Create a secure directory
    secure_temp_dir="$1"
    update_config FNRETURN ""

    # Remove existing directory if it exists (optional)
    $rm -rf "$secure_temp_dir"

    # Create the directory with restrictive permissions
    create_secure_dir "$secure_temp_dir"

    # create temp and update
    temp_folder=$($mktemp -d -p "$secure_temp_dir")

    update_config FNRETURN "$temp_folder"
}

function resourced_tempdata_vars {
    local config_file="$SCRIPT_DIR/os-update-tool.config"

    # Iterate through each line and identify variables dependent on tempdata
    while IFS= read -r line; do
        # Skip empty lines or comments
        [[ -z "$line" || "$line" =~ ^# ]] && continue

        # Extract variable name and value using regex
        if [[ "$line" =~ ^([a-zA-Z_][a-zA-Z0-9_]*)=(.+)$ ]]; then
            var_name="${BASH_REMATCH[1]}"
            var_value="${BASH_REMATCH[2]}"

            # If the value references tempdata, update the variable
            # shellcheck disable=SC2016
            if [[ "$var_value" == *'$TEMPDATA'* ]]; then
                update_config "$var_name" "$(eval echo "$var_value")"
            fi
        fi
    done < "$config_file"
}

function cleanName {
    update_config FNRETURN ""
    local name="$1"
    # Remove all leading non-alphanumeric characters and spaces
    name=$(echo "$name" | $sed 's/^[^a-zA-Z0-9]*//')
    update_config FNRETURN "$name"
}
