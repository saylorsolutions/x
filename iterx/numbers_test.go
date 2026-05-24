package iterx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSum(t *testing.T) {
	numbers := []int{2, 3, 7}
	assert.Equal(t, 12.0, Sum(Select(numbers))) //nolint:testifylint // This is deterministically scaled.
}

func TestAverage(t *testing.T) {
	numbers := []int{2, 3, 7}
	assert.Equal(t, 4.0, Average(Select(numbers))) //nolint:testifylint // This is deterministically scaled.
}

func TestStdDev(t *testing.T) {
	numbers := []int{2, 2, 4, 4}
	assert.Equal(t, 1.0, StdDev(Select(numbers))) //nolint:testifylint // This is deterministically scaled.
}
