// Package token contains code for generating, validating, and decoding JWTs.
package token

import (
	"errors"
	"time"
)

// keyName is the name of the environment variable to retrieve the JWT signing
// key from.
const keyName = "JWTKEY"

// ErrInvalid means that the given token is invalid.
var ErrInvalid = errors.New("invalid token")

// EncodeFunc defines a type that can be used to encode a token.
type EncodeFunc[T any] func(time.Time, T) (string, error)

// DecodeFunc defines a type that can be used to decode a JWT.
type DecodeFunc[T any] func(string) (T, error)
