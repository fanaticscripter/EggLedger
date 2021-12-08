package main

import (
	"math"
	"time"
)

func timeToUnix(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}

func unixToTime(t float64) time.Time {
	sec, dec := math.Modf(t)
	return time.Unix(int64(sec), int64(dec*1e9))
}
