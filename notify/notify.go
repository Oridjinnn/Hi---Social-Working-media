package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

func FireOS(title, body, iconPath string) error {
	return beeep.Notify(title, body, iconPath)
}

func FormatConnectionTitle() string {
	return "HI — New Connection"
}

func FormatConnectionBody(actorUsername, projectName string) string {
	return fmt.Sprintf("@%s connected to your %s signal", actorUsername, projectName)
}