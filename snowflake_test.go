package snowflake

import (
	"fmt"
	"testing"
)

func TestSnowFlake_Next(t *testing.T) {
	for i := 0; i < 100000; i++ {
		fmt.Println(Next())
	}
}

func BenchmarkSnowFlake_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Next()
	}
}
