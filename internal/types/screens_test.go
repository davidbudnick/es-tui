package types

import "testing"

func TestScreenString(t *testing.T) {
	cases := []struct {
		s    Screen
		want string
	}{
		{ScreenConnections, "Connections"},
		{ScreenIndices, "Indices"},
		{ScreenSearch, "Search"},
		{Screen(999), "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Fatalf("%v: got %q want %q", tc.s, got, tc.want)
		}
	}
	// exercise all known screens once
	for s := ScreenConnections; s <= ScreenCatAPI; s++ {
		if s.String() == "Unknown" {
			t.Fatalf("screen %d should have a name", s)
		}
	}
}
