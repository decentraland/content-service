package metrics

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestToMillis(t *testing.T) {
	d := time.Duration(1000000000)

	assert.Equal(t, float64(1), d.Seconds())

	m := toMillis(d)

	assert.Equal(t, float64(1000), m)

	now := time.Now()

	future := now.Add(time.Duration(10) * time.Second)

	result := future.Sub(now)

	m = toMillis(result)

	assert.Equal(t, float64(10000), m)
}
