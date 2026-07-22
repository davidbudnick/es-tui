package testutil

import (
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestMockGetIndexDetail(t *testing.T) {
	m := &MockES{IndexDetail: types.IndexInfo{Name: "forced"}}
	idx, err := m.GetIndex("anything")
	AssertNoError(t, err)
	AssertEqual(t, idx.Name, "forced")
}
