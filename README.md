# AB OS Update Tool
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Lint Status](https://https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml/badge.svg)](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml)
[![Unit Test Status](https://https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/unit-test.yml/badge.svg)](https://[github.com/intel-innersource/os.linux.tiberos.ab-update](https://github.com/open-edge-platform/edge-microvisor-update-tool)/actions/workflows/unit-test.yml)

The OS Update Tool (UT) is a utility used by INMB/PUA agents for Day 2 operations. UT provides interfaces to assist INMB/PUA agents in performing OS image A/B swaps. Its functionalities include OS image downloading, flashing to an alternate partition, and updating the next boot configuration.

___
## How to use the AB UT for OS AB swapping

The default location for the AB Update Tool will be /usr/bin/os-update-tool.sh

### Step 1. List available function
```
cd /usr/bin
```
```
sudo ./os-update-tool.sh -h
```
**output:**
```
os-update-tool ver-1.7

Usage: sudo os-update-tool.sh [-r] [-v] [-a] [-c] [-w] [-u string] [-s string] [-h] [--debug]

Options:
  -r      Restore to previous boot.
  -v      Display current active partition.
  -a      Apply updated image as next boot.
  -c      Commit Updated image as default boot.
  -w      Write rootfs partition.
  -u      Define update image source path.
  -s      Define sha256 checksum value for the update image.
  -h      Display this help message.
  --debug Executes with debug log.
```

### Step 2. Write to inactive partition

##### The logic will look for partition marked with rootfs
```
sudo ./os-update-tool.sh -w -u /mnt/tiber-readonly-1.0.20240920.1734.raw.xz -s e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

### Step 3: Apply image as next boot
```
sudo ./os-update-tool.sh -a
```

### Step 4: reboot the device
```
reboot
```

### Step 5: Commit image as a default OS to boot if everything works fine after rebooted from previous step
```
sudo ./os-update-tool.sh -c
```

___
## Other useful functions

### 1. Restore previous boot
```
sudo ./os-update-tool.sh -r
```

### 2. Display current boot partition and its image UUID
```
sudo ./os-update-tool.sh -v
```

### 3. Enables Debug to any command by adding "--debug"
```
sudo ./os-update-tool.sh -v --debug
```

### 4. Enable developer mode by adding "--dev"

Currently, dev mode will auto create user for the newly installed OS.

```
sudo ./os-update-tool.sh -w -u /mnt/tiber-readonly-1.0.20240920.1734.raw.xz -s e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 --dev
```
## License Information

Edge Microvisor Update Tool is open source and licensed under the MIT License. See the [LICENSE](./LICENSE) file for more details.
