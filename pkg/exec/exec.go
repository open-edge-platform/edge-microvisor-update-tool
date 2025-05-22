package exec

import (
	"fmt"
	"os/exec"
	"strings"
)

var commandMap = map[string]string{
	"ls":          "/usr/bin/ls",
	"lsblk":       "/usr/bin/lsblk",
	"grep":        "/usr/bin/grep",
	"cut":         "/usr/bin/cut",
	"head":        "/usr/bin/head",
	"blkid":       "/usr/sbin/blkid",
	"find":        "/usr/bin/find",
	"sfdisk":      "/usr/sbin/sfdisk",
	"basename":    "/usr/bin/basename",
	"cryptsetup":  "/usr/sbin/cryptsetup",
	"veritysetup": "/usr/sbin/veritysetup",
	"mkdir":       "/usr/bin/mkdir",
	"mount":       "/usr/bin/mount",
	"umount":      "/usr/bin/umount",
	"rm":          "/usr/bin/rm",
	"dmsetup":     "/usr/sbin/dmsetup",
	"sync":        "/usr/bin/sync",
	"tune2fs":     "/usr/sbin/tune2fs",
	"dirname":     "/usr/bin/dirname",
	"df":          "/usr/bin/df",
	"tail":        "/usr/bin/tail",
	"sleep":       "/usr/bin/sleep",
	"bash":        "/usr/bin/bash",
	"chmod":       "/usr/bin/chmod",
	"mktemp":      "/usr/bin/mktemp",
	"sed":         "/usr/bin/sed",
	"xz":          "/usr/bin/xz",
	"gzip":        "/usr/bin/gzip",
	"flock":       "/usr/bin/flock",
	"sha256sum":   "/usr/bin/sha256sum",
	"bootctl":     "/usr/bin/bootctl",
	"uniq":        "/usr/bin/uniq",
	"dd":          "/usr/bin/dd",
	"fdisk":       "/usr/sbin/fdisk",
	"losetup":     "/usr/sbin/losetup",
	"cp":          "/usr/bin/cp",
	"findmnt":     "/usr/bin/findmnt",
	"chroot":      "/usr/sbin/chroot",
	"e2fsck":      "/usr/sbin/e2fsck",
	// Add more mappings as needed
}

func getActualCmd(key string) (string, error) {
	// Look up the command in the map
	if cmd, exists := commandMap[key]; exists {
		return cmd, nil
	}

	//todo: the full path birth date vs os-release file birthdate, newer = invalid
	return "", fmt.Errorf("command not found or not allowed: %s", key)
}

// ExecuteCommand runs a Linux command passed as a string and returns the output and error if any.
func ExecuteCommand(cmdKey string, args ...string) (string, error) {

	// Use getActualCmd to retrieve the actual command
	command, err := getActualCmd(cmdKey)
	if err != nil {
		return "", err
	}

	// Validate the command to avoid potential security risks
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Ensure the command is a full path
	if !strings.HasPrefix(command, "/") {
		return "", fmt.Errorf("command must be a full path, got: %s", command)
	}

	// Create the command
	cmd := exec.Command(command, args...)

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v, output: %s", err, string(output))
	}

	return string(output), nil
}
