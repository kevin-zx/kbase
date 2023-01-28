package ktime

import "time"

func GetDateRange(rangeDays int, deltaDays int) []time.Time {
	end := time.Now().AddDate(0, 0, deltaDays)
	start := end.AddDate(0, 0, -rangeDays)
	return []time.Time{start, end}
}
