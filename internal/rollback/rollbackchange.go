package apply

import "fmt"

func RollbackChange() (int, error) {

	//TODO: osId is identified by finding the OS ID that is not currently booted.
	// Steps:
	// 1. Retrieve the list of available boot entries using a command like "bootctl list".
	// 2. Determine the current boot OS ID by checking the active boot entry.
	// 3. Identify the OS ID that is not the current boot OS ID.
	// 4. Use the identified OS ID to set the default boot entry with a command like "bootctl set-default <osId>".
	//if found, then set the default boot to the id

	osId := "dummy.uki"
	fmt.Println("bootctl set-default ", osId)

	return 0, nil
}
