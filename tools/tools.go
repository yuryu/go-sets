//go:build go_mod_tidy_check

// Package tools tracks tool dependencies that are not needed to build.
package tools

import (
	_ "honnef.co/go/tools/cmd/staticcheck"
)
