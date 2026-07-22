package testutil

import "testing"

type recTB struct {
	fataled bool
}

func (r *recTB) Helper()                           {}
func (r *recTB) Fatalf(format string, args ...any) { r.fataled = true }
func (r *recTB) Fatal(args ...any)                 { r.fataled = true }

func TestAssertFailBranches(t *testing.T) {
	r := &recTB{}
	AssertEqual(r, 1, 2)
	if !r.fataled {
		t.Fatal("AssertEqual")
	}
	r = &recTB{}
	AssertNoError(r, errString("e"))
	if !r.fataled {
		t.Fatal("AssertNoError")
	}
	r = &recTB{}
	AssertError(r, nil)
	if !r.fataled {
		t.Fatal("AssertError")
	}
	r = &recTB{}
	AssertTrue(r, false, "x")
	if !r.fataled {
		t.Fatal("AssertTrue")
	}
}
