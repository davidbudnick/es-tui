package testutil

import "testing"

// TB is the subset of testing.TB used by assert helpers.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
	Fatal(args ...any)
}

// AssertEqual fails if got != want.
func AssertEqual[T comparable](t TB, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// AssertNoError fails if err is not nil.
func AssertNoError(t TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails if err is nil.
func AssertError(t TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertTrue fails if cond is false.
func AssertTrue(t TB, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Fatal(msg)
	}
}

// Ensure *testing.T satisfies TB.
var _ TB = (*testing.T)(nil)
