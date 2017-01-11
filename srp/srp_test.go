package srp

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b(bs ...byte) []byte {
	return bs
}

func repeat(s string, n int) string {
	var b = new(bytes.Buffer)
	for i := 0; i < n; i++ {
		b.Write([]byte(s))
	}
	return b.String()
}

func TestExchange(t *testing.T) {
	var fakeU = "Luke"
	var fakeP = "theforce"
	var group = Group2048
	db := NewMapLookup()
	db.Add(fakeU, fakeP, group)

	server := NewServerInstance(db)
	client, err := NewClient(fakeU, fakeP, &RFCGroups)
	require.Nil(t, err)

	smat, err := server.KeyExchange(fakeU)
	require.Nil(t, err)

	keyC, A, err := client.KeyExchange(smat)
	require.Nil(t, err)

	keyS, err := server.Key(A)
	require.Nil(t, err)
	require.Equal(t, keyC, keyS)

}

func TestMakeX(t *testing.T) {
	var validU = "Chewbacca"
	var validP = "Falcon"
	for i, tt := range []struct {
		User   string
		Pwd    string
		Salt   []byte // nil == random
		IError bool
		XError bool
	}{
		{"", "", nil, true, true},
		{repeat("i", 500), "", nil, true, true},
		{validU, "", nil, true, true},
		{validU, repeat("p", 500), nil, true, true},
		{validU, validP, nil, false, false}, // good
		{validU, validP, random(10), false, true},
		{validU, validP, random(SaltSize), false, false},
	} {
		if tt.Salt == nil {
			tt.Salt = random(SaltSize)
		}
		inner, err := makeInner(tt.User, tt.Pwd)
		if tt.IError && err == nil {
			t.Errorf("%d: inner should have returned error", i)
			continue
		} else if !tt.IError && err != nil {
			t.Errorf("%d: inner should not have returned error", i, err)
			continue
		} else if err != nil {
			continue
		}

		_, err2 := makeX(inner, tt.Salt)
		if tt.XError && err2 == nil {
			t.Errorf("%d: should have returned error", i)
		} else if !tt.XError && err2 != nil {
			t.Errorf("%d: should not have returned error %s", i, err2)
		}

	}
}

func TestPad(t *testing.T) {
	for _, tt := range []struct {
		Slice    []byte
		Length   int
		Panic    bool
		Expected []byte
	}{
		{b(0x01, 0x02), 4, false, b(0x00, 0x00, 0x01, 0x02)},
		{b(0x01, 0x02), 2, false, b(0x01, 0x02)},
		{b(0x01, 0x02, 0x03), 2, true, nil},
	} {
		testPad(t, tt.Slice, tt.Expected, tt.Length, tt.Panic)
	}
}

func testPad(t *testing.T, s, exp []byte, l int, p bool) {
	defer func() {
		if e := recover(); e != nil {
			if !p {
				t.Error(e)
			}
		} else if p {
			t.Error("Should have panic'd")
		}
	}()
	res := pad(s, l)
	assert.Equal(t, exp, res)
}

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
		} else if p {
			t.Error("should have panic'd")
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
