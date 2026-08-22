package main

import "testing"

func TestAddHeaderKeepsLeadingBuildConstraint(t *testing.T) {
	const header = "// license\n"
	const src = `//go:build e2e

// old license

package tests
`
	const want = `//go:build e2e

// license

package tests
`

	got := string(addHeader([]byte(src), header))
	if got != want {
		t.Fatalf("addHeader() = %q, want %q", got, want)
	}

	if gotAgain := string(addHeader([]byte(got), header)); gotAgain != want {
		t.Fatalf("addHeader() is not idempotent: got %q, want %q", gotAgain, want)
	}
}
