package slogx

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"strings"
	"testing"
)

func TestDedupeHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(NewDedupeHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	log = log.With("testkey", 1)
	log = log.With("testkey", 2)
	log = log.With("testkey", 3)
	log = log.With("testkey", 4)
	log.Info("Test")
	handler := log.Handler().(*DedupeHandler)
	assert.Equal(t, 1, strings.Count(buf.String(), "testkey"))
	assert.Len(t, handler.attrs, 1)
	assert.Equal(t, int64(4), handler.attrs[0].Value.Int64())
	assert.Len(t, handler.attrSet, 1)
}

func TestDedupeHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(NewDedupeHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	log = log.With("testkey", 1)
	log = log.With("testkey", 2)
	log = log.WithGroup("group")
	log = log.With("groupkey", 1)
	log = log.With("groupkey", 2)
	log.Info("Test")
	handler := log.Handler().(*DedupeHandler)
	assert.Equal(t, 1, strings.Count(buf.String(), "testkey"))
	assert.Equal(t, 1, strings.Count(buf.String(), "group.groupkey"))
	assert.Len(t, handler.attrs, 2)
	assert.Len(t, handler.attrSet, 2)
}

func TestDedupeHandler_NilImpl(t *testing.T) {
	assert.Panics(t, func() {
		NewDedupeHandler(nil)
	})
}
