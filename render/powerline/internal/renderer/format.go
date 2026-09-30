// Package renderer provides status line rendering.
package renderer

import (
	"strconv"
	"time"
)

// Pace classification thresholds, in percentage points away from an even burn.
const (
	// minutesPerHour converts an hour count into minutes.
	minutesPerHour int = 60
	// hoursPerDay converts a day count into hours.
	hoursPerDay int = 24
)

// FormatDuration renders a duration in the most compact readable form.
// It favours a single significant unit: a status line has no room for
// "2 hours 10 minutes 4 seconds", and the seconds carry no decision value.
//
// Params:
//   - d: duration to render
//
// Returns:
//   - string: compact duration such as "45m", "2h10", "3d"
func FormatDuration(d time.Duration) string {
	// A non-positive duration has already elapsed
	if d <= 0 {
		return "now"
	}

	minutes := int(d.Minutes())
	// Under an hour, minutes alone are the clearest unit
	if minutes < minutesPerHour {
		return strconv.Itoa(minutes) + "m"
	}

	hours := minutes / minutesPerHour
	// Under a day, hours and minutes give the useful precision
	if hours < hoursPerDay {
		remaining := minutes % minutesPerHour
		// A whole number of hours needs no minute part
		if remaining == 0 {
			return strconv.Itoa(hours) + "h"
		}
		return strconv.Itoa(hours) + "h" + pad2(remaining)
	}

	days := hours / hoursPerDay
	remaining := hours % hoursPerDay
	// A whole number of days needs no hour part
	if remaining == 0 {
		return strconv.Itoa(days) + "d"
	}
	return strconv.Itoa(days) + "d" + strconv.Itoa(remaining) + "h"
}

// FormatTokens renders a token count in a compact k/M form.
//
// Params:
//   - tokens: token count
//
// Returns:
//   - string: compact count such as "812" , "103k" or "1.2M"
func FormatTokens(tokens int) string {
	const (
		thousand = 1000
		million  = 1000 * 1000
	)
	// Below a thousand the raw count is already short
	if tokens < thousand {
		return strconv.Itoa(tokens)
	}
	// Below a million, thousands keep it to at most four characters
	if tokens < million {
		return strconv.Itoa(tokens/thousand) + "k"
	}
	whole := tokens / million
	tenths := (tokens % million) / (million / 10)
	// A whole number of millions needs no decimal
	if tenths == 0 {
		return strconv.Itoa(whole) + "M"
	}
	return strconv.Itoa(whole) + "." + strconv.Itoa(tenths) + "M"
}

// pad2 left-pads a value below ten with a zero.
//
// Params:
//   - value: value to pad
//
// Returns:
//   - string: two-character representation
func pad2(value int) string {
	// Single digits need a leading zero to read as minutes
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
