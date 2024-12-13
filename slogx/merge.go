package slogx

import (
	"context"
	"errors"
	"log/slog"
)

var _ slog.Handler = (*handlerJoiner)(nil)

type handlerJoiner struct {
	a, b slog.Handler
}

func (h *handlerJoiner) Enabled(ctx context.Context, level slog.Level) bool {
	return h.a.Enabled(ctx, level) || h.b.Enabled(ctx, level)
}

func (h *handlerJoiner) Handle(ctx context.Context, record slog.Record) error {
	aerr := h.a.Handle(ctx, record)
	berr := h.b.Handle(ctx, record)
	return errors.Join(aerr, berr)
}

func (h *handlerJoiner) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.a = h.a.WithAttrs(attrs)
	h.b = h.b.WithAttrs(attrs)
	return h
}

func (h *handlerJoiner) WithGroup(name string) slog.Handler {
	h.a = h.a.WithGroup(name)
	h.b = h.b.WithGroup(name)
	return h
}

// MergeHandlers will merge many [slog.Handler] into one for a single interface for both.
func MergeHandlers(a, b slog.Handler, others ...slog.Handler) slog.Handler {
	joined := &handlerJoiner{a, b}
	if len(others) > 0 {
		return MergeHandlers(joined, others[0], others[1:]...)
	}
	return joined
}
