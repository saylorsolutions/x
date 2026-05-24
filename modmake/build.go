package main

import (
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck
)

func main() {
	b := NewBuild()
	b.LintLatest().
		Enable("testifylint", "thelper", "testableexamples").
		EnableSecurityScanning()

	b.Execute()
}
