package model

import "time"

// Now returns the current UTC time. Separated for testability.
var Now = func() time.Time {
	return time.Now().UTC()
}
