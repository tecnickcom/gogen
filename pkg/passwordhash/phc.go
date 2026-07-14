package passwordhash

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	// phcSeparator delimits the top-level PHC fields and also marks a stored value
	// as PHC: the base64 alphabet never contains '$', so a leading '$' is an
	// unambiguous signal that a hash is PHC rather than base64-encoded JSON.
	phcSeparator = "$"

	// phcCostSeparator delimits the comma-separated cost parameters (m, t, p).
	phcCostSeparator = ","

	// phcSegments is the number of fields a versioned Argon2 PHC string splits
	// into on '$': a leading empty field, the algorithm, the version, the cost
	// parameters, the salt, and the key.
	phcSegments = 6

	// phcCostFields is the number of comma-separated cost parameters (m, t, p).
	phcCostFields = 3

	phcVersionPrefix = "v="
	phcMemoryPrefix  = "m="
	phcTimePrefix    = "t="
	phcThreadsPrefix = "p="
)

// marshalPHC renders data as a PHC string. The parameters originate from a
// freshly minted [Hashed] whose [Params.Algo] is always argon2id, so the output
// is a well-formed Argon2 PHC string that any compliant implementation can read.
func marshalPHC(data *Hashed) string {
	prm := data.Params

	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		prm.Algo, prm.Version, prm.Memory, prm.Time, prm.Threads,
		base64.RawStdEncoding.EncodeToString(data.Salt),
		base64.RawStdEncoding.EncodeToString(data.Key),
	)
}

// unmarshalPHC parses a PHC string into a [Hashed]. It reconstructs the key and
// salt lengths from the decoded byte lengths (PHC does not store them
// explicitly) and returns [ErrInvalidHashData] for any malformed field, so a
// forged or corrupt string is rejected before it reaches argon2. The numeric
// bounds enforced elsewhere ([Params.validateVerifyData]) still apply to the result.
//
// Only canonical strings are accepted: newline characters (which Go's base64
// decoder would otherwise silently skip) and non-canonical trailing base64 bits
// are rejected, so byte-distinct stored strings can never decode to the same
// salt and key.
func unmarshalPHC(hash string) (*Hashed, error) {
	if strings.ContainsAny(hash, "\r\n") {
		return nil, fmt.Errorf("%w: PHC string contains newline characters", ErrInvalidHashData)
	}

	// decodeHash only routes strings with a leading '$' here, so SplitN always
	// yields an empty first field; the segment count is what distinguishes a
	// well-formed versioned Argon2 PHC string from a malformed one. The split is
	// capped one past the expected count so an adversarial string full of '$'
	// cannot allocate an arbitrarily large slice just to be rejected.
	parts := strings.SplitN(hash, phcSeparator, phcSegments+1)
	if len(parts) != phcSegments {
		return nil, fmt.Errorf("%w: malformed PHC string", ErrInvalidHashData)
	}

	params, err := parsePHCParams(parts[1], parts[2], parts[3])
	if err != nil {
		return nil, err
	}

	salt, err := decodePHCBytes(parts[4], "salt")
	if err != nil {
		return nil, err
	}

	key, err := decodePHCBytes(parts[5], "key")
	if err != nil {
		return nil, err
	}

	params.SaltLen = uint32(len(salt))
	params.KeyLen = uint32(len(key))

	return &Hashed{Params: params, Salt: salt, Key: key}, nil
}

// decodePHCBytes decodes the salt or key segment of a PHC string using the
// canonical encoding mandated by the PHC specification: standard base64
// alphabet, no padding, and strict trailing-bit validation so every byte
// sequence has exactly one accepted encoding.
func decodePHCBytes(segment, field string) ([]byte, error) {
	b, err := base64.RawStdEncoding.Strict().DecodeString(segment)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid PHC %s encoding: %w", ErrInvalidHashData, field, err)
	}

	return b, nil
}

// parsePHCParams parses the algorithm, version, and cost segments of a PHC
// string into a [Params]. SaltLen and KeyLen are left zero here; the caller
// fills them from the decoded salt and key lengths.
func parsePHCParams(algo, versionSeg, costSeg string) (*Params, error) {
	version, err := parsePHCUint(versionSeg, phcVersionPrefix, 8)
	if err != nil {
		return nil, err
	}

	// Capped one past the expected count for the same reason as the '$' split in
	// unmarshalPHC: a comma-flooded segment must not allocate a large slice just
	// to be rejected by the count check below.
	cost := strings.SplitN(costSeg, phcCostSeparator, phcCostFields+1)
	if len(cost) != phcCostFields {
		return nil, fmt.Errorf("%w: malformed PHC cost parameters %q", ErrInvalidHashData, costSeg)
	}

	memory, err := parsePHCUint(cost[0], phcMemoryPrefix, 32)
	if err != nil {
		return nil, err
	}

	time, err := parsePHCUint(cost[1], phcTimePrefix, 32)
	if err != nil {
		return nil, err
	}

	threads, err := parsePHCUint(cost[2], phcThreadsPrefix, 8)
	if err != nil {
		return nil, err
	}

	return &Params{
		Algo:    algo,
		Version: uint8(version),
		Time:    uint32(time),
		Memory:  uint32(memory),
		Threads: uint8(threads),
	}, nil
}

// parsePHCUint strips prefix from segment and parses the remainder as an unsigned
// integer that must fit in bitSize bits. It returns [ErrInvalidHashData] when the
// prefix is absent or the value is not a valid in-range number, so a malformed or
// oversized field never panics or overflows the destination parameter.
func parsePHCUint(segment, prefix string, bitSize int) (uint64, error) {
	rest, ok := strings.CutPrefix(segment, prefix)
	if !ok {
		return 0, fmt.Errorf("%w: expected %q in PHC segment %q", ErrInvalidHashData, prefix, segment)
	}

	value, err := strconv.ParseUint(rest, 10, bitSize)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid number in PHC segment %q: %w", ErrInvalidHashData, segment, err)
	}

	return value, nil
}
