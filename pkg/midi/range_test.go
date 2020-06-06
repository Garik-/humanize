package midi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRange(t *testing.T) {
	var n int64 = 480
	r := newMyRange(0, n)

	var x int64 = 960
	for {
		if r.contains(x) == true {
			break
		}

		if x > n {
			r.stepBy(int(x / n))
		} else {
			r.stepBy(1)
		}
	}

	assert.Equal(t, 2, r.cnt)
	assert.Equal(t, int64(960), r.lowerBound)
	assert.Equal(t, int64(1440), r.upperBound)
	assert.Equal(t, 2, r.position())
}
