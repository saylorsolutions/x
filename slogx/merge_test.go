package slogx

import (
	"github.com/stretchr/testify/assert"
	"log/slog"
	"strings"
	"testing"
)

func TestMergeHandlers(t *testing.T) {
	var (
		bufA, bufB, bufC strings.Builder
	)
	log := slog.New(MergeHandlers(
		slog.NewTextHandler(&bufA, &slog.HandlerOptions{}),
		slog.NewTextHandler(&bufB, &slog.HandlerOptions{}),
		slog.NewTextHandler(&bufC, &slog.HandlerOptions{}),
	))
	log.Info("A message", "test", "test")
	a, b, c := bufA.String(), bufB.String(), bufC.String()
	assert.NotEmpty(t, a)
	assert.Equal(t, a, b)
	assert.Equal(t, b, c)
}
