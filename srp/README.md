# Secure Remote Password protocol

This is a library implementing the SRP protocol as defined in [RFC
2945](https://www.rfc-editor.org/rfc/rfc2945.txt) (partially) and in the [RFC
5054](https://tools.ietf.org/html/rfc5054) for the TLS integration.

## Usage

 1. Initialization phase 

The server has to register all username and password that it is expected to
server. The password are not stored directly, but the hash with salt. The
example here use an implemented local memory store.

```go
// enter the creds
db := NewMapLookup()
db.Add("Serge","Karamazov",Group4096)

server := NewServerInstance(db)
```
 
 2. Exchange

*server*:
```go
// give the username to the server
mat,err := server.KeyExchange("Serge",nil)
if err != nil {
    panic(err)
}
```

*client*:
```go
// give the material to the client
client,err := NewClient("Serge","Karamazov"
if err != nil {
    panic(err)
}
keyC,A,err := client.KeyExchange(mat)
if err != nil {
    panic(err)
}
// keyC is the actual KEY derived :)
```

*server*:
```
keyS,err := server.Key(A)
if err != nil {
    panic(err)
}

// keyS is the actual KEY derived :)
```

## Differences with the RFC 5054

+ The hash function used here is SHA256 instead of SHA-1
+ The number of groups allowed is reduced (1024 and 2048 bit size are dropped) 

## Open questions

+ field elements size ? a & b. Set to RandSize = 64

## License

MIT License. See LICENSE file.
