package passwordhash

import (
	"fmt"
	"strings"

	"github.com/tecnickcom/nurago/pkg/encode"
)

// Format identifies a serialization of the stored hash string. [WithFormat]
// selects which one [Params.PasswordHash] emits and which ones
// [Params.PasswordNeedsRehash] treats as current. Verification auto-detects the
// format of every stored value, so any [Params] can read hashes written in
// either format regardless of configuration.
type Format uint8

const (
	// FormatJSON is the default self-describing base64-encoded JSON format
	// documented in the package overview. It is the zero value, so it is what a
	// [Params] uses unless [WithFormat] selects otherwise.
	FormatJSON Format = iota

	// FormatPHC is the PHC string format shared by Argon2 implementations across
	// languages and runtimes:
	//
	//	$argon2id$v=19$m=65536,t=3,p=4$<base64 salt>$<base64 key>
	//
	// Choose it for interoperability with external tooling that reads or writes
	// PHC strings (for example PHP's password_hash, Python's argon2-cffi/passlib,
	// or the Argon2 reference CLI). The salt and key are encoded with the standard
	// base64 alphabet without padding, as required by the PHC specification.
	// Only argon2id PHC strings can be verified; the package overview documents
	// the exact accepted envelope.
	FormatPHC
)

// normalizeFormat maps any Format value onto a supported one: FormatPHC stays
// itself and everything else (including unknown values) falls back to
// FormatJSON, so an out-of-range emit value can never select an unsupported
// serialization.
func normalizeFormat(f Format) Format {
	if f == FormatPHC {
		return FormatPHC
	}

	return FormatJSON
}

// formatBit returns the bit representing f in an accepted-formats mask.
// f must be normalized first; the two supported formats map to distinct bits.
func formatBit(f Format) uint8 {
	return 1 << f
}

// detectFormat reports the serialization a stored hash string uses: a leading
// '$' marks a PHC string (the base64 alphabet used by the JSON format never
// produces one), anything else is base64-encoded JSON.
func detectFormat(hash string) Format {
	if strings.HasPrefix(hash, phcSeparator) {
		return FormatPHC
	}

	return FormatJSON
}

// decodeHash decodes a stored hash string into a [Hashed], auto-detecting the
// serialization with [detectFormat]. It is the read-side counterpart to
// [Params.serialize].
func decodeHash(hash string) (*Hashed, error) {
	if detectFormat(hash) == FormatPHC {
		return unmarshalPHC(hash)
	}

	data := &Hashed{}

	err := encode.Deserialize(hash, data)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to decode the hash string: %w", ErrInvalidHashData, err)
	}

	return data, nil
}

// serialize renders a freshly minted [Hashed] using the configured [Format].
// PHC is a pure in-memory render that cannot fail; the JSON path can surface an
// encoding error.
func (ph *Params) serialize(data *Hashed) (string, error) {
	if ph.format == FormatPHC {
		return marshalPHC(data), nil
	}

	return encode.Serialize(data) //nolint:wrapcheck
}

// acceptsFormat reports whether f is among the formats PasswordNeedsRehash
// treats as current (see [WithFormat]). f must come from [detectFormat], which
// only returns supported values, so the bit shift is always in range.
func (ph *Params) acceptsFormat(f Format) bool {
	return ph.acceptedFormats&formatBit(f) != 0
}
