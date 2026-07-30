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
// The caller decides how to report a failure; this function never writes to
// the terminal itself.
func SendNotification(title, message string) error {
	if err := beeep.Notify(title, message, iconData); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}

// PlaySound plays a short, discrete alert sound using the OS's default beep
// mechanism (a single system beep, not a media file). Frequency and
// duration are left at zero so each platform substitutes its own short
// default (e.g. ~200ms on Linux, ~500ms on Windows, a single system beep on
// macOS), rather than gomodoro picking arbitrary values.
func PlaySound() error {
	if err := beeep.Beep(0, 0); err != nil {
		return fmt.Errorf("play sound: %w", err)
	}
	return nil
}
