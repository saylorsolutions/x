package main

import (
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck
)

func main() {
	b := NewBuild()
	b.LintLatest().
		Enable("testifylint", "thelper", "testableexamples", "perfsprint", "nolintlint", "noctx", "modernize", "mnd", "godox", "gocyclo", "gocritic").
		EnableSecurityScanning()
	b.Execute()
}
