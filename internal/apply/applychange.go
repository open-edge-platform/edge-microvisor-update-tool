package apply

import "fmt"

func ApplyChange() (int, error) {

	// TODO: Identify the OS ID by locating a file in the /boot directory
	// with a .bak extension. Remove the .bak extension from the filename
	// and set the next boot to use the corresponding uki file.
	// Example command: bootctl set-oneshot dummy.uki

	osId := "dummy.uki"
	fmt.Println("bootctl set-oneshot ", osId)

	return 0, nil
}
