package adapt

import (
	"errors"
)

var (
	ErrNoDefault    = errors.New("no default adaptation for object")
	ErrNoAdaptation = errors.New("no adaptation from object to trait")
)
