package slogx

import (
	"context"
	"log/slog"
)

type attrSet map[string]bool

func (s attrSet) dupe() attrSet {
	dupe := attrSet{}
	for k, v := range s {
		dupe[k] = v
	}
	return dupe
}

var _ slog.Handler = (*DedupeHandler)(nil)

type DedupeHandler struct {
	group   string
	attrSet attrSet
	attrs   []slog.Attr
	impl    slog.Handler
}

func NewDedupeHandler(impl slog.Handler) slog.Handler {
	if impl == nil {
		panic("nil implementing handler")
	}
	return &DedupeHandler{
		impl: impl,
	}
}

func (s *DedupeHandler) prefix() string {
	if len(s.group) == 0 {
		return ""
	}
	return s.group + "."
}

func (s *DedupeHandler) dupe() *DedupeHandler {
	cp := &DedupeHandler{
		group:   s.group,
		attrSet: s.attrSet.dupe(),
		attrs:   s.attrs,
		impl:    s.impl,
	}
	return cp
}

func (s *DedupeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.impl.Enabled(ctx, level)
}

func (s *DedupeHandler) Handle(ctx context.Context, record slog.Record) error {
	var addtlAttrs []slog.Attr
	if record.NumAttrs() > 0 {
		addtlAttrs = make([]slog.Attr, record.NumAttrs())
		i := 0
		record.Attrs(func(attr slog.Attr) bool {
			addtlAttrs[i] = attr
			i++
			return true
		})
		record = slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
		s = s.WithAttrs(addtlAttrs).(*DedupeHandler)
	}
	return s.impl.WithAttrs(s.attrs).Handle(ctx, record)
}

func (s *DedupeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return s
	}
	cp := s.dupe()
	prefix := cp.prefix()
	for _, attr := range attrs {
		newKey := prefix + attr.Key
		if cp.attrSet[newKey] {
			for i := 0; i < len(cp.attrs); i++ {
				if cp.attrs[i].Key == newKey {
					attr.Key = newKey
					cp.attrs[i] = attr
					break
				}
			}
			continue
		}
		attr.Key = newKey
		cp.attrs = append(cp.attrs, attr)
		cp.attrSet[newKey] = true
	}
	return cp
}

func (s *DedupeHandler) WithGroup(name string) slog.Handler {
	cp := s.dupe()
	cp.group = cp.prefix() + name
	return cp
}
