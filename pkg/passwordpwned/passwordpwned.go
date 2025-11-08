/*
Package passwordpwned allows you to verify if a password has been pwned
(compromised) in a data breach.

The checks are performed by using the k-anonymity model of the HIBP service API
(https://haveibeenpwned.com/API/v3#PwnedPasswords).

The client transmits only the first 5 characters of the SHA-1 hash of the
password to query the HIBP service API.
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

// IsPwnedPassword returns true if the password has been found pwned.
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
