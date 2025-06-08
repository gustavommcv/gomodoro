package notify

import "os/exec"

// TODO
// Start and continue buttons in the notification dialog box
func Notify(message string) {
	// TODO
	// Cross plataform notifications
	cmd := exec.Command("notify-send", message)
	cmd.Run()
}
