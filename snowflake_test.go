package xid

import (
	"testing"
)

func TestSnowFlake_GetId(t *testing.T) {
	var sf, _ = NewSnowFlake()

	for i := 0; i < 10000000; i++ {
		sf.Next()
	}
}

func BenchmarkNewSnowFlake(b *testing.B) {
	var sf, _ = NewSnowFlake(WithDataCenter(1), WithMachine(1))
	for i := 0; i < b.N; i++ {
		sf.Next()
	}
}
