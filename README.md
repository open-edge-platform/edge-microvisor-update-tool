# OS Update Tool
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Lint Status](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml/badge.svg)](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/lint-sh.yml)
[![Unit Test Status](https://github.com/open-edge-platform/edge-microvisor-update-tool/actions/workflows/unit-test.yml/badge.svg)](https://[github.com/intel-innersource/os.linux.tiberos.ab-update](https://github.com/open-edge-platform/edge-microvisor-update-tool)/actions/workflows/unit-test.yml)

The OS Update Tool provides a command-line interface (CLI) to manage Immutability OS image updates. Its core functionalities include OS raw image extraction, multiple Unified Kernel Imaage managememnt, flashing to an alternate boot partition, and updating the next boot configuration. The OS Update Tool delivers a command-line interface (CLI) for the streamlined management of Immutability OS image updates. Its essential capabilities include the extraction of OS raw images, management of multiple Unified Kernel Images (UKIs), flashing images to an alternate boot partition, precise configuration of the next system boot and error recovery.

___

## Using the OS Update Tool for Immutable OS Updates

The default location for the OS Update Tooll will be /usr/bin/os-update-tool.sh

### Step 1. List available function

```bash
cd /usr/bin
```

```bash
sudo ./os-update-tool.sh -h
```

**output:**

```bash
os-update-tool ver-2.9

Usage: sudo os-update-tool.sh [-v] [-a] [-c] [-w] [-u string] [-s string] [-h] [--debug]

Options:
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
### 1. Display current boot partition and its image UUID

```bash
sudo ./os-update-tool.sh -v
```

### 2. Enables Debug by adding "--debug"

```bash
sudo ./os-update-tool.sh -v --debug
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
