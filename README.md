# tlsrp

This package is a fork of golang/crypto/tls with SRP support. It implements the
[RFC 5054](https://tools.ietf.org/html/rfc5054).

## Disclaimer

**WARNING**: This implementation has not been peer-reviewed at all, and may be
full of exploitable bugs. USE AT YOUR OWN RISK.

## Usage

First, you must `go get -u github.com/nikkolasg/tlsrp`. Here are the basics
steps you can setup a password protected communication channel between two parties using SRP:

 1. Server part:

```go
import "github.com/nikkolasg/tlsrp"
import "github.com/nikkolasg/tlsrp/srp"

// register user / password pairs ( hashed + salted ) 
m := srp.NewMapLookup()
if err := m.Add("Patrick", "Bialès", srp.Group4096); err != nil {
	t.Fatal(err)
}


listConf := tlsrp.SRPConfigServer(m)
listener,err := tlsrp.Listener("tcp","127.0.0.1:8080",listConf)
if err != nil {
    panic(err)
}
```

 2. Client connection

```go

clientConf, err := tlsrp.SRPConfigUser("Patrick","Bialès")
if err != nil {
    panic(err)
}
conn, err := tlsrp.Dial("tcp","127.0.0.1:8080",clientConf)
if err != nil {
    panic(err)
}

// from now on, you can use the conn as usual. If there is an error in the SRP
// handshake, it will show up.
```

## Implementation

I tried to touch as little as possible to the tls implementation. I found out
that it was more difficult than expected. All components are deeply tighted
together. However, all tests are passing.

All checks from the RFC are implemented as well as the "hide wrong username"
[feature](https://tools.ietf.org/html/rfc5054#page-6).

## Inspiration

The idea came from the [Magic
Wormhole](https://github.com/warner/magic-wormhole) tool by Brian Warner.
Unfortunately, I needed a secure transport protocol out of it, and in go. Since
I did not want to roll up any home-made tls, I chose to follow an already
existing scheme with proven security and where an RFC already exists.
