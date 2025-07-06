package iterx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSum(t *testing.T) {
	numbers := []int{2, 3, 7}
	assert.Equal(t, 12.0, Sum(Select(numbers)))
}

func TestAverage(t *testing.T) {
	numbers := []int{2, 3, 7}
	assert.Equal(t, 4.0, Average(Select(numbers)))
}

func TestStdDev(t *testing.T) {
	numbers := []int{2, 2, 4, 4}
	assert.Equal(t, 1.0, StdDev(Select(numbers)))
}
