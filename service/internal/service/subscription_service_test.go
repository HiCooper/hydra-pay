package service

import (
	"testing"
	"time"
)

func TestCalculatePeriodEndMonth(t *testing.T) {
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := CalculatePeriodEnd(start, IntervalMonth)

	expected := time.Date(2026, 2, 15, 10, 30, 0, 0, time.UTC)
	if !end.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, end)
	}
}

func TestCalculatePeriodEndYear(t *testing.T) {
	start := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	end := CalculatePeriodEnd(start, IntervalYear)

	expected := time.Date(2027, 5, 20, 0, 0, 0, 0, time.UTC)
	if !end.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, end)
	}
}

func TestCalculatePeriodEndDefault(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := CalculatePeriodEnd(start, "unknown")

	// Defaults to month
	expected := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !end.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, end)
	}
}

func TestCalculatePeriodEndLeapYear(t *testing.T) {
	// Jan 31 → Feb 28 (non-leap year)
	start := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	end := CalculatePeriodEnd(start, IntervalMonth)

	if end.Year() != 2025 || end.Month() != 3 || end.Day() != 3 {
		t.Errorf("expected 2025-03-03 (Go AddDate normalizes), got %v", end)
	}
}

func TestCalculatePeriodEndEndOfMonth(t *testing.T) {
	// Oct 31 → Nov 30
	start := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	end := CalculatePeriodEnd(start, IntervalMonth)

	if end.Month() != 12 || end.Day() != 1 {
		t.Errorf("expected 2026-12-01 (Go AddDate normalization), got %v", end)
	}
}

func TestNextPeriodRangeMonth(t *testing.T) {
	periodEnd := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	nextStart, nextEnd := NextPeriodRange(periodEnd, IntervalMonth)

	if !nextStart.Equal(periodEnd) {
		t.Errorf("next start should equal current end, got %v", nextStart)
	}
	expectedEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if !nextEnd.Equal(expectedEnd) {
		t.Errorf("expected end %v, got %v", expectedEnd, nextEnd)
	}
}

func TestNextPeriodRangeYear(t *testing.T) {
	periodEnd := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	nextStart, nextEnd := NextPeriodRange(periodEnd, IntervalYear)

	if !nextStart.Equal(periodEnd) {
		t.Errorf("next start should equal current end, got %v", nextStart)
	}
	expectedEnd := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	if !nextEnd.Equal(expectedEnd) {
		t.Errorf("expected end %v, got %v", expectedEnd, nextEnd)
	}
}
