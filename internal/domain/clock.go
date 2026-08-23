package domain

import "time"

type Clock interface{ Now() time.Time }
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type FixedClock struct{ Value time.Time }

func (c FixedClock) Now() time.Time   { return c.Value }
func DateKey(value time.Time) string  { return value.UTC().Format("2006-01-02") }
func MonthKey(value time.Time) string { return value.UTC().Format("2006-01") }
func EndOfDay(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
}
func StartOfDay(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
func SameBusinessDay(a, b time.Time) bool { return DateKey(a) == DateKey(b) }
