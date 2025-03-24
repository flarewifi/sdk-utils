package boot

import (
	"fmt"
	"time"
)

func constructDoneMsg(start time.Time) string {
	var (
		duration = time.Since(start)
		minutes  = duration / time.Minute
		seconds  = (duration % time.Minute) / time.Second

		minStr = "minute"
		secStr = "second"
	)

	if minutes > 1 {
		minStr = "minutes"
	}
	if seconds > 1 {
		secStr = "seconds"
	}

	return fmt.Sprintf("Done booting in %d %v and %d %v", minutes, minStr, seconds, secStr)
}
