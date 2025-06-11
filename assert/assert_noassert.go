//go:build noassert

package assert

func Disable() {
	// No op
}

func Enable() {
	// No op
}

func True(label string, result bool) {
	// No op
}

func TrueFunc(label string, assertion func() bool) {
	// No op
}

func NotEmpty(label string, val any) {
	// No op
}
