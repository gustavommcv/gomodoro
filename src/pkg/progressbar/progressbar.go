package progressbar

import (
	"fmt"
	"io"
	"math"
	"strings"
)

func ProgressBar(writer io.Writer, progress float64, total float64) {
	percent := int(math.Round(100 * progress / total))
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	filled := percent / 10
	empty := 10 - filled

	fmt.Fprintf(writer, "%d%% [%s%s]", percent,
		strings.Repeat("█", filled),
		strings.Repeat("-", empty))
}
