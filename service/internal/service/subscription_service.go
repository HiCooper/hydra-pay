package service

import (
	"time"
)

// SubscriptionInterval constants
const (
	IntervalMonth = "month"
	IntervalYear  = "year"
)

// CalculatePeriodEnd returns the end of a billing period given a start time and interval.
func CalculatePeriodEnd(start time.Time, interval string) time.Time {
	switch interval {
	case IntervalYear:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

// NextPeriodRange returns the next billing period (start, end) after the current period end.
func NextPeriodRange(currentPeriodEnd time.Time, interval string) (time.Time, time.Time) {
	start := currentPeriodEnd
	return start, CalculatePeriodEnd(start, interval)
}
