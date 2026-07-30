package beep

import (
	_ "embed"
	"fmt"

	"github.com/gen2brain/beeep"
)

//go:embed assets/logo.png
var iconData []byte

func init() {
	beeep.AppName = "Gomodoro"
}

// SendNotification shows a desktop notification with the Gomodoro icon. The
// caller decides how to report a failure; this function never writes to the
// terminal itself.
func SendNotification(title, message string) error {
	if err := beeep.Alert(title, message, iconData); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}
