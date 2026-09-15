// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

//nolint:depguard
package json

import (
	"io"
	"slices"

	"encoding/json/jsontext"
	json "encoding/json/v2"
)

var allowInvalid []json.Options = slices.Clip([]json.Options{jsontext.AllowInvalidUTF8(true)})

func Marshal(in any, opts ...json.Options) (out []byte, err error) {
	if len(opts) == 0 {
		opts = allowInvalid
	} else {
		opts = append(allowInvalid, opts...)
	}
	return json.Marshal(in, opts...)
}

func MarshalEncode(out *jsontext.Encoder, in any, opts ...json.Options) (err error) {
	if len(opts) == 0 {
		opts = allowInvalid
	} else {
		opts = append(allowInvalid, opts...)
	}
	return json.MarshalEncode(out, in, opts...)
}

func MarshalWrite(out io.Writer, in any, opts ...json.Options) (err error) {
	if len(opts) == 0 {
		opts = allowInvalid
	} else {
		opts = append(allowInvalid, opts...)
	}
	return json.MarshalWrite(out, in, opts...)
}

func MarshalIndent(in any, prefix, indent string) (out []byte, err error) {
	if prefix == "" && indent == "" {
		// WithIndentPrefix and WithIndent imply multiline output, so skip them.
		return Marshal(in)
	}
	return Marshal(in, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

func MarshalIndentWrite(out io.Writer, in any, prefix, indent string) (err error) {
	if prefix == "" && indent == "" {
		// WithIndentPrefix and WithIndent imply multiline output, so skip them.
		return MarshalWrite(out, in)
	}
	return MarshalWrite(out, in, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

func Unmarshal(in []byte, out any, opts ...json.Options) (err error) {
	return json.Unmarshal(in, out, opts...)
}

func UnmarshalDecode(in *jsontext.Decoder, out any, opts ...json.Options) (err error) {
	return json.UnmarshalDecode(in, out, opts...)
}

func UnmarshalRead(in io.Reader, out any, opts ...json.Options) (err error) {
	return json.UnmarshalRead(in, out, opts...)
}

func AllowDuplicateNames(allow bool) json.Options {
	return jsontext.AllowDuplicateNames(allow)
}

func Deterministic(v bool) json.Options {
	return json.Deterministic(v)
}

func WithIndent(indent string) json.Options {
	return jsontext.WithIndent(indent)
}

func NewDecoder(r io.Reader) *jsontext.Decoder {
	return jsontext.NewDecoder(r)
}

type (
	Value           = jsontext.Value
	Kind            = jsontext.Kind
	UnmarshalerFrom = json.UnmarshalerFrom
	MarshalerTo     = json.MarshalerTo
	Decoder         = jsontext.Decoder
	Encoder         = jsontext.Encoder
)

var (
	BeginObject = jsontext.BeginObject
	EndObject   = jsontext.EndObject
	Null        = jsontext.Null
	BeginArray  = jsontext.BeginArray
	EndArray    = jsontext.EndArray
)
