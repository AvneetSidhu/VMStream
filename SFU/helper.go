package sfu

import (
	"time"
)

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefUint(i *uint16) uint16 {
	if i == nil {
		return 0
	}
	return *i
}

func toNTPTime(t time.Time) uint64 {
	const ntpEpoch = 2208988800 // NTP epoch starts on 1900-01-01
	seconds := uint64(t.Unix()) + ntpEpoch
	fraction := uint64(t.Nanosecond()) * 0xFFFFFFFF / 1e9 // Convert nanoseconds to fraction
	return (seconds << 32) | fraction
}