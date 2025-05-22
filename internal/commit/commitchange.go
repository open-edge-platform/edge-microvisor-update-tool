package apply

import "fmt"

func CommitChange() (int, error) {

	// Identify the current boot OS ID by querying the bootloader configuration
	// (e.g., using `bootctl` or reading from /boot/loader/entries/).
	// Once the current OS ID is determined, set the next boot ID using
	// `bootctl set-default <osId>` or an equivalent command.

	osId := "dummy.uki"
	fmt.Println("bootctl set-default ", osId)

	return 0, nil
}
