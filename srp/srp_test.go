package srp

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandom(t *testing.T) {
	for _, tt := range []struct {
		Reader io.Reader
		Length int
		Panic  bool
	}{
		{Rand, 64, false},
		{Rand, -1, true},
		{Rand, 1001, true},
		{strings.NewReader(""), 100, true},
		{strings.NewReader("hi"), 10, true},
	} {
		testRandom(t, tt.Reader, tt.Length, tt.Panic)
	}
}

func testRandom(t *testing.T, r io.Reader, l int, p bool) {
	old := Rand
	Rand = r
	defer func() {
		Rand = old
		if e := recover(); e != nil {
			if !p {
				t.Error(e)
			}
		}
	}()
	out := random(l)
	assert.Equal(t, l, len(out))
}

func TestMapLookup(t *testing.T) {
	db := NewMapLookup()
	var group = Group2048
	var uname = "ender"
	var pwd = "game"
	db.Add(uname, pwd, group)

	info, ok := db.Fetch(uname)
	assert.True(t, ok)
	assert.True(t, info.Group.Equal(group))

	info, ok = db.Fetch("random")
	assert.Nil(t, info)
	assert.False(t, ok)
}
