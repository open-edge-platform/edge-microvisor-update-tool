# AB OS Update Tool
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Lint Status](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml/badge.svg)](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml)
[![Unit Test Status](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/unit-test.yml/badge.svg)](https://[github.com/intel-innersource/os.linux.tiberos.ab-update](https://github.com/open-edge-platform/edge-microvisor-update-tool)/actions/workflows/unit-test.yml)

The OS Update Tool (UT) is a utility used by the Inband Management Agent/Platform
Update Agent for Day 2 operations. The UT provides interfaces to assist INMB/PUA
agents in performing OS image A/B updates. Its functionalities include OS image
downloading, flashing to an alternate partition, and updating the next boot
configuration.

___

## How to use the AB UT for OS AB swapping

The default location for the AB Update Tool will be /usr/bin/os-update-tool.sh

### Step 1. List available function

```bash
cd /usr/bin
```

```bash
sudo ./os-update-tool.sh -h
```

**output:**

```bash
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

The logic will look for partition marked with rootfs

```bash
sudo ./os-update-tool.sh -w -u /mnt/tiber-readonly-1.0.20240920.1734.raw.xz -s e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

### Step 3: Apply image as next boot

```bash
sudo ./os-update-tool.sh -a
```

### Step 4: Reboot the device

```bash
reboot
```

### Step 5: Commit image as a default image

If everything works fine after rebooted in previous step, commit the image.

```bash
sudo ./os-update-tool.sh -c
```

___

## Additional functionality

### 1. Restore previous boot

```bash
sudo ./os-update-tool.sh -r
```

### 2. Display current boot partition and its image UUID

```bash
sudo ./os-update-tool.sh -v
```

### 3. Enables Debug by adding "--debug"

```bash
sudo ./os-update-tool.sh -v --debug
```

### 4. Enable Developer mode by adding "--dev"

Currently, dev mode will auto create user for the newly installed OS.

```bash
sudo ./os-update-tool.sh -w -u /mnt/tiber-readonly-1.0.20240920.1734.raw.xz -s e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 --dev
```

## Getting Help

If you encounter bugs, have feature requests, or need assistance,
[file a GitHub Issue](https://github.com/open-edge-platform/edge-microvisor-update-tool/issues).

Before submitting a new report, check the existing issues to see if a similar one has not
been filed already. If no matching issue is found, feel free to file the issue as described
in the [contribution guide](./CONTRIBUTING.md).

For security-related concerns, please refer to [SECURITY.md](./SECURITY.md).

## License Information

Edge Microvisor Update Tool is open source and licensed under the MIT License.
See the [LICENSE](./LICENSE) file for more details.

## Contributing

As an open-source project, Edge Microvisor Update Tool always looks for
community-driven improvements. If you are interested in making the product even
better, see how you can help in the [contribution guide](./CONTRIBUTING.md).
