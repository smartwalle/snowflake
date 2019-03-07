package snowflake

import (
	"fmt"
	"testing"
	"time"
)

func TestSnowFlake_Next(t *testing.T) {

	var sf, _ = New(WithTimeOffset(time.Date(2019, time.March, 7, 0, 0, 0, 0, time.UTC)))

	for i := 0; i < 100000; i++ {
		fmt.Println(sf.Next())
	}

}
