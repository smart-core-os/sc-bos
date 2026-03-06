package api

import (
	"fmt"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

var Commands = []string{
	"LEFT",
	"RIGHT",
	"UP",
	"DOWN",
	"ZOOM_IN",
	"ZOOM_OUT",
	"LEFT_UP",
	"LEFT_DOWN",
	"RIGHT_UP",
	"RIGHT_DOWN",
	"FOCUS_NEAR",
	"FOCUS_FAR",
	"IRIS_ENLARGE",
	"IRIS_REDUCE",
}

func MovementToCommand(mov *traits.PtzMovement) string {
	var zoom, pan, tilt string
	if mov.Direction.Zoom != 0 {
		if mov.Direction.Zoom > 0 {
			zoom = "ZOOM_OUT"
		} else {
			zoom = "ZOOM_IN"
		}
	} else if mov.Direction.Pan != 0 {
		if mov.Direction.Pan > 0 {
			pan = "RIGHT"
		} else {
			pan = "LEFT"
		}
	} else if mov.Direction.Tilt != 0 {
		if mov.Direction.Tilt > 0 {
			tilt = "UP"
		} else {
			tilt = "DOWN"
		}
	}
	if zoom != "" {
		return zoom
	}
	if pan != "" && tilt != "" {
		return fmt.Sprintf("%s_%s", pan, tilt)
	}
	if pan != "" {
		return pan
	}
	if tilt != "" {
		return tilt
	}
	return ""
}
