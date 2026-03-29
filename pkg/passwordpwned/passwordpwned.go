/*
Package passwordpwned checks whether a password has appeared in a known data
breach, using the Have I Been Pwned (HIBP) Pwned Passwords API v3
(https://haveibeenpwned.com/API/v3#PwnedPasswords).

# Problem

Storing or transmitting a plain-text password to a third-party service to check
whether it has been compromised would itself be a security vulnerability.
Applications that want to reject breached passwords during registration or
credential-change flows need a safe, privacy-preserving way to perform the
check without revealing the password or its full hash to any external service.

# Solution

This package implements the HIBP k-anonymity model: only the first 5 hex
characters of the SHA-1 hash are sent over the network. The API returns all
hash suffixes that share that 5-character prefix (typically 800–1 000 entries
due to Add-Padding). The full hash match is resolved entirely on the client
side, so the actual password and its complete hash never leave the process.

Create a client with [New] and call [Client.IsPwnedPassword]:

	c, err := passwordpwned.New()
	if err != nil {
		log.Fatal(err)
	}

	pwned, err := c.IsPwnedPassword(ctx, password)
	if err != nil {
		log.Fatal(err)
	}

	if pwned {
		return errors.New("password has been compromised in a data breach, please choose another")
	}

# Features

  - Privacy-preserving k-anonymity: only a 5-character hash prefix is
    transmitted; the full hash and the original password never leave the caller.
  - Add-Padding support: requests include the Add-Padding header so all
    responses contain 800–1 000 entries regardless of real match count,
    preventing traffic-analysis attacks.
  - Brotli decoding: responses are decoded with brotli compression out of
    the box, reducing bandwidth for high-volume callers.
  - Automatic retries: a configurable [httpretrier]-backed retry policy
    handles transient network errors safely for read-only requests.
  - Functional options: [WithURL], [WithTimeout], [WithHTTPClient],
    [WithUserAgent], [WithRetryAttempts], and [WithRetryDelay] let callers
    customize every aspect of the HTTP client without subclassing.
  - Testable by design: [HTTPClient] is an interface, making it trivial to
    inject a mock HTTP client in unit tests.

# Security Note

The SHA-1 algorithm is used solely because the HIBP API requires it. The
password is hashed locally; only the 5-character prefix is sent over TLS.
This package does not store, log, or persist the password or its hash.

# Benefits

This package provides a single-function integration with the industry-standard
HIBP Pwned Passwords service, handling hashing, network communication,
compression, padding, and retries so application code stays focused on
business logic.
*/
package passwordpwned

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	brotli "github.com/aperturerobotics/go-brotli-decoder"
)

// IsPwnedPassword reports whether password has appeared in a known data breach.
//
// The check is performed using the HIBP k-anonymity model:
//  1. The SHA-1 hash of password is computed locally.
//  2. Only the first 5 hex characters of the hash are sent to the HIBP API.
//  3. The API returns all matching hash suffixes (padded to 800–1 000 entries).
//  4. The full hash is matched client-side; the password never leaves the process.
//
// Returns (true, nil) if the password is found in the breach database.
// Returns (false, nil) if the password is not found or its breach count is zero.
// Returns (false, err) on any network, encoding, or unexpected HTTP error.
//
//nolint:nonamedreturns
func (c *Client) IsPwnedPassword(ctx context.Context, password string) (pwned bool, err error) {
	c.hashObj.Reset()

	_, werr := io.WriteString(c.hashObj, password)
	if werr != nil {
		return false, fmt.Errorf("unable to hash password: %w", werr)
	}

	hash := strings.ToUpper(hex.EncodeToString(c.hashObj.Sum(nil)))

	r, nerr := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/"+rangePath+"/"+hash[:5], nil)
	if nerr != nil {
		return false, fmt.Errorf("create request: %w", nerr)
	}

	r.Header.Set("User-Agent", c.userAgent)
	r.Header.Set("Accept-Encoding", "br") // Responses are brotli-encoded.
	r.Header.Set("Add-Padding", "true")   // All responses will contain between 800 and 1,000 results regardless of the number of hash suffixes returned by the service.

	hr, herr := c.newHTTPRetrier()
	if herr != nil {
		return false, fmt.Errorf("create retrier: %w", herr)
	}

	resp, derr := hr.Do(r)
	if derr != nil {
		return false, fmt.Errorf("execute request: %w", derr)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	reader := brotli.NewReader(resp.Body)

	data, rerr := io.ReadAll(reader)
	if rerr != nil {
		return false, fmt.Errorf("error decoding brotli response: %w", rerr)
	}

	idx := bytes.Index(data, []byte(hash[5:]))

	// A password is not pwned if the hash suffix is not found
	// or the recurrence is zero.
	if (idx < 0) || (data[idx+36] == '0') {
		return false, nil
	}

	return true, nil
}
