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

func SendNotification(title, message string) {
	err := beeep.Alert(title, message, iconData)
	if err != nil {
		fmt.Printf("Error in notification: %v\n", err)
	}
}
