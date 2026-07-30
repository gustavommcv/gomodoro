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

// SendNotification shows a silent desktop notification with the Gomodoro
// icon. It never plays a sound itself — beeep.Alert bundles an OS sound
// into the notification, which would make the notify and sound alerts
// impossible to control independently, so this uses beeep.Notify instead.
// The sound alert lives in package sound, not here: it needs a real audio
// backend (see that package's doc comment for why), which is a different
// concern from showing a notification bubble.
// The caller decides how to report a failure; this function never writes to
// the terminal itself.
func SendNotification(title, message string) error {
	if err := beeep.Notify(title, message, iconData); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}
