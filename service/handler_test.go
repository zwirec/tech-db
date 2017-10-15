package service

import (
	"testing"
	"time"
)

func BenchmarkBytesToTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b := []byte{'0', '2', '-', '0', '1', '-', '2', '0', '0', '6','0', '2', '-', '0', '1', '-', '2', '0', '0', '6'}
		time.Parse("02-01-2006", string(b))
	}
}

func BenchmarkStringToTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := "02-01-2006"
		time.Parse("02-01-2006", s)
	}
}