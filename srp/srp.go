package srp

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"
)

// The hashing function used for the SRP protocol. NOTE: It might be adding an
// extra security layer to convert it to Argon2 for example later.
var HashFunc = sha256.New

// SaltSize is the size of the salt to generate in bytes.
// Taken from tlslite implementation.
const SaltSize = 16

// RandSize is the size of the random numbers generated by the client and server
// Taken from tlslite implementation.
const RandSize = 32

// Rand is the reader used to generate randomness for the salt and the points in
// the exchange. By default, it is set to `crypto/rand.Reader`.
var Rand io.Reader

// MinUserSize is the minimum size an username must be, inclusive.
const MinUserSize = 3

// MaxUserSize is the maximum size an username can be, inclusive.
const MaxUserSize = 255

var zero = big.NewInt(0)

func init() {
	Rand = rand.Reader
}

// Verifier contains  H(salt || (username:password)). It is suitable for
// writing/reader from most of encoding schemes (json).
type Verifier struct {
	// the actual group element. Encoded as big.Int which by default uses
	// big endianness format.
	Hash []byte
	// Salt
	Salt []byte
}

// NewVerifier returns a new verifier out of the username and the related
// password. The password can be discarded after as there is no need to store
// it. http://tools.ietf.org/html/rfc5054#section-2.4
func NewVerifier(username, password string, group Group) (*Verifier, error) {
	salt := random(SaltSize)
	inner, err := makeInner(username, password)
	if err != nil {
		return nil, err
	}
	x, err := makeX(inner, salt)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Exp(group.G, x, group.N)
	return &Verifier{
		Hash: v.Bytes(),
		Salt: salt,
	}, nil
}

type Client struct {
	user    string
	allowed Groups
	inner   []byte // H( I || ':' || P)
	a       *big.Int
	A       *big.Int
	m       *ServerMaterial
	key     []byte
}

// NewClient returns a client that is able to proceed on the SRP protocol. It
// consumes the password into a hash function so there's no storage of it.
func NewClient(username, password string, allowedGroups *Groups) (*Client, error) {
	var grs Groups
	if allowedGroups == nil {
		grs = RFCGroups
	} else {
		grs = *allowedGroups
	}
	inner, err := makeInner(username, password)
	return &Client{
		user:    username,
		inner:   inner,
		allowed: grs,
	}, err
}

// Material uses the server materials to generate the random component A from
// the client  as in 2.6 https://tools.ietf.org/html/rfc5054#page-8
// It returns the shared key, the public part A to send to server, or
// errInvalidB if the value provided by the // server is wrong (== 0).
func (c *Client) KeyExchange(m *ServerMaterial) (key, A []byte, err error) {
	if !c.allowed.Contains(m.Group) {
		return nil, nil, errors.New("Unknown group given by server")

	}

	if len(m.B) != m.Group.Len() {
		return nil, nil, errors.New("srp invalid B size")
	} else if len(m.Salt) != SaltSize {
		return nil, nil, errors.New("srp invalid salt size")
	}

	B := new(big.Int).SetBytes(m.B)
	if B.Mod(B, m.Group.N).Cmp(zero) == 0 {
		return nil, nil, errors.New("Invalid B public element from server")
	}
	c.a = new(big.Int).SetBytes(random(RandSize))
	c.A = new(big.Int).Exp(m.Group.G, c.a, m.Group.N)
	c.m = m
	A = c.A.Bytes()

	u := makeU(c.A, B, m.Group.Len())
	x, err := makeX(c.inner, m.Salt)
	if err != nil {
		return nil, nil, err
	}
	// key = (B - (k * g^x)) ^ (a + (u * x)) % N
	k := makeK(m.Group)
	base := new(big.Int).Exp(m.Group.G, x, m.Group.N)
	base.Mul(k, base).Sub(B, base)
	exp := new(big.Int).Mul(u, x)
	exp.Add(c.a, exp)
	key = base.Exp(base, exp, m.Group.N).Bytes()
	c.key = key
	return
}

func (c *Client) Username() string {
	return c.user
}

// Key returns a copy of the key
func (c *Client) Key() []byte {
	return append([]byte{}, c.key...)
}

type Lookup interface {
	Fetch(username string) (*UserInfo, bool)
}

type UserInfo struct {
	Verifier *Verifier
	Group    Group
}

// ServerInstance is a struct following the protocol FOR ONE USER
type ServerInstance struct {
	db   Lookup
	info *UserInfo
	b    *big.Int
	B    *big.Int
	key  *big.Int
}

func NewServerInstance(lookup Lookup) *ServerInstance {
	return &ServerInstance{
		db: lookup,
	}
}

type ServerMaterial struct {
	Salt  []byte
	B     []byte
	Group Group
}

var ErrUnknownUser = errors.New("username provided is not known")

// KeyExchange proceeds to the key exchange part from the server's point of
// view. It computes B = k * v + g^b % N and returns the information needed by
// the Client to pursue. fakseSalt is used when the username is invalid: in that
// case, the server will simulate computation as usual in order to avoid timing
// attacks. If fakeSalt is nil, it directly returns and do NO DO this fake
// computations. In any cases, if the username is invalid, it returns
// ErrUnknownUser.
func (s *ServerInstance) KeyExchange(username string, fakeSalt []byte) (*ServerMaterial, error) {
	var err error
	info, ok := s.db.Fetch(username)
	if !ok {
		// simulate computations to avoid timing attacks when caller wants to
		// hide that the username is invalid
		err = ErrUnknownUser
		if fakeSalt == nil {
			return nil, err
		}
		info = &UserInfo{
			Verifier: &Verifier{
				Salt: fakeSalt,
				Hash: random(32),
			},
			Group: Group4096,
		}
	}
	s.info = info
	group := info.Group
	s.b = toInt(random(RandSize))
	commit := new(big.Int).Exp(group.G, s.b, group.N)
	k := makeK(group)
	v := toInt(info.Verifier.Hash)
	left := new(big.Int).Mul(k, v)
	s.B = commit.Add(left, commit).Mod(commit, group.N)
	return &ServerMaterial{
		Salt:  info.Verifier.Salt,
		Group: info.Group,
		B:     s.B.Bytes(),
	}, err
}

// Key returns the shared key given the A public client's information or an
// error if A is wrong or suspicious.
func (s *ServerInstance) Key(A []byte) ([]byte, error) {
	group := s.info.Group
	if len(A) != group.Len() {
		return nil, errors.New("Material A wrong length")
	}
	aint := toInt(A)
	if new(big.Int).Mod(aint, group.N).Cmp(zero) == 0 {
		return nil, errors.New("Material A suspicious")
	}
	u := makeU(aint, s.B, group.Len())
	v := toInt(s.info.Verifier.Hash)
	// key = (A * v^u) ^ b % N
	base := new(big.Int).Exp(v, u, group.N)
	base.Mul(base, aint).Exp(base, s.b, group.N)
	s.key = new(big.Int).Set(base)
	return base.Bytes(), nil
}

// u = H( PAD(A) || PAD(B) )
func makeU(A, B *big.Int, len int) *big.Int {
	return toInt(hash(pad(A.Bytes(), len), pad(B.Bytes(), len)))
}

// k = H( N || PAD(G) )
func makeK(gr Group) *big.Int {
	return toInt(hash(gr.N.Bytes(), pad(gr.G.Bytes(), gr.Len())))
}

func makeInner(username, password string) ([]byte, error) {
	cont := username + ":" + password
	if !utf8.ValidString(cont) {
		return nil, errors.New("username:password not valid utf8")
	}
	if len(username) == 0 || len(password) >= 256 {
		return nil, errors.New("username invalid length")
	}
	if len(password) == 0 || len(password) >= 256 {
		return nil, errors.New("password invalid length")
	}

	return hash([]byte(cont)), nil
}

// x = H( salt || H( username || ':' || password ) )
func makeX(inner, salt []byte) (*big.Int, error) {
	if len(salt) != SaltSize {
		return nil, errors.New("salt invalid length")
	}
	return toInt(hash(salt, inner)), nil
}

type MapLookup map[string]*UserInfo

func NewMapLookup() *MapLookup {
	m := MapLookup(make(map[string]*UserInfo))
	return &m
}

// Add an user to the database with the
func (m *MapLookup) Add(uname, password string, group Group) error {

	v, err := NewVerifier(uname, password, group)
	if err != nil {
		return err
	}
	info := &UserInfo{
		Verifier: v,
		Group:    group,
	}
	(*m)[uname] = info
	return nil
}

func (m *MapLookup) Fetch(username string) (*UserInfo, bool) {
	i, o := (*m)[username]
	return i, o
}

func hash(s ...[]byte) []byte {
	h := HashFunc()
	for _, slice := range s {
		h.Write(slice)
	}
	return h.Sum(nil)
}

func random(length int) []byte {
	// yeah constant whatever, just don't make me do too big things
	if length <= 0 || length >= 1000 {
		panic(fmt.Sprintf("random() with length %s given", length))
	}
	var buff = make([]byte, length)
	n, err := Rand.Read(buff)
	if err != nil {
		panic(err)
	} else if n != length {
		panic(fmt.Sprintf("random()	took only %d / %d bytes", n, length))
	}
	return buff
}

func pad(s []byte, n int) []byte {
	var l = len(s)
	if l == n {
		return s
	} else if l > n {
		panic("pad called with bigger slice than allowed")
	}
	var rem = n - l
	var s2 = make([]byte, n)
	copy(s2[rem:], s)
	s = nil
	return s2
}

func toInt(s []byte) *big.Int {
	return new(big.Int).SetBytes(s)
}
