/*
Package passwordpwned allows to verify if a password was pwned
via the haveibeenpwned.com (HIBP) service API.

Ref.: https://haveibeenpwned.com/API/v3#PwnedPasswords
*/
package passwordpwned

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Vonage/gosrvlib/pkg/logging"
	brotli "github.com/aperturerobotics/go-brotli-decoder"
)

// IsPwnedPassword returns true if the password has been found pwned.
func (c *Client) IsPwnedPassword(ctx context.Context, password string) (bool, error) {
	c.hashObj.Reset()

	_, err := io.WriteString(c.hashObj, password)
	if err != nil {
		return false, fmt.Errorf("unable to hash password: %w", err)
	}

	hash := strings.ToUpper(hex.EncodeToString(c.hashObj.Sum(nil)))

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/"+rangePath+"/"+hash[:5], nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	r.Header.Set("user-agent", c.userAgent)
	r.Header.Set("Accept-Encoding", "br") // Responses are brotli-encoded.
	r.Header.Set("Add-Padding", "true")   // All responses will contain between 800 and 1,000 results regardless of the number of hash suffixes returned by the service.

	hr, err := c.newHTTPRetrier()
	if err != nil {
		return false, fmt.Errorf("create retrier: %w", err)
	}

	resp, err := hr.Do(r) //nolint:bodyclose
	if err != nil {
		return false, fmt.Errorf("execute request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error closing response body")

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	reader := brotli.NewReader(resp.Body)

	data, err := io.ReadAll(reader)
	if err != nil {
		return false, fmt.Errorf("error decoding brotli response: %w", err)
	}

	idx := bytes.Index(data, []byte(hash[5:]))

	// A password is not pwned if the hash suffix is not found
	// or the recurrence is zero.
	if (idx < 0) || (data[idx+36] == '0') {
		return false, nil
	}

	return true, nil
}
