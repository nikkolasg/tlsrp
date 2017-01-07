package srp

type Client struct {
	Username string
	Password string
	Groups   []Group
}

type Lookup interface {
	// Fetch returns the verifier, the sal and the group used for this user
	// name. If the name does not exists, it must returns an error.
	Fetch(user string) (v, s []byte, grp Group, err error)
}

type Server struct {
	Lookup
	// used in hmac for generating fake salts, for hiding existence of account
	SaltKey []byte
	// size of fake salts (<= 64), should match what is used for valid accounts
	SaltSize []byte
}
