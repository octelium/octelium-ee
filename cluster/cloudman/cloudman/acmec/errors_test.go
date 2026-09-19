// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/acme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseRetryAfter(t *testing.T) {
	assert.Equal(t, 30*time.Second, parseRetryAfter("30"))
	assert.Equal(t, time.Duration(0), parseRetryAfter(""))
	assert.Equal(t, time.Duration(0), parseRetryAfter("0"))
	assert.Equal(t, time.Duration(0), parseRetryAfter("-10"))
	assert.Equal(t, time.Duration(0), parseRetryAfter("not-a-date"))

	{
		at := time.Now().Add(time.Hour).UTC()
		delay := parseRetryAfter(at.Format(time.RFC3339))
		assert.True(t, delay > 55*time.Minute && delay <= time.Hour, "%s", delay)
	}

	{
		at := time.Now().Add(time.Hour).UTC()
		delay := parseRetryAfter(at.Format(http.TimeFormat))
		assert.True(t, delay > 55*time.Minute && delay <= time.Hour, "%s", delay)
	}
}

func TestGetRetryAfter(t *testing.T) {
	{
		err := &acme.RateLimitedError{
			ProblemDetails: &acme.ProblemDetails{
				Type: acme.RateLimitedErr,
			},
			RetryAfter: "120",
		}

		assert.Equal(t, 2*time.Minute, getRetryAfter(errors.Wrap(err, "Could not obtain")))
	}

	{
		at := time.Now().Add(2 * time.Hour).UTC()
		err := errors.Errorf(
			"acme: error: 429 :: too many certificates already issued, retry after %s",
			at.Format(time.RFC3339))

		delay := getRetryAfter(err)
		assert.True(t, delay > 110*time.Minute && delay <= 2*time.Hour, "%s", delay)
	}

	assert.Equal(t, time.Duration(0), getRetryAfter(errors.Errorf("Could not do it")))
}

func TestIsBadNonce(t *testing.T) {
	assert.True(t, isBadNonce(&acme.NonceError{
		ProblemDetails: &acme.ProblemDetails{
			Type: acme.BadNonceErr,
		},
	}))

	assert.True(t, isBadNonce(errors.Wrap(&acme.ProblemDetails{
		Type: acme.BadNonceErr,
	}, "Could not obtain")))

	assert.False(t, isBadNonce(errors.Errorf("Could not do it")))
}

func TestIsRateLimited(t *testing.T) {
	assert.True(t, isRateLimited(&acme.RateLimitedError{
		ProblemDetails: &acme.ProblemDetails{Type: acme.RateLimitedErr},
	}))

	assert.True(t, isRateLimited(&acme.ProblemDetails{
		HTTPStatus: http.StatusTooManyRequests,
	}))

	assert.True(t, isRateLimited(errors.Errorf("Error: too many requests")))
	assert.False(t, isRateLimited(errors.Errorf("Could not do it")))
}

func TestIsAccountDoesNotExist(t *testing.T) {
	assert.True(t, isAccountDoesNotExist(&acme.ProblemDetails{
		Type: "urn:ietf:params:acme:error:accountDoesNotExist",
	}))

	assert.True(t, isAccountDoesNotExist(errors.Errorf("acme: account does not exist")))
	assert.False(t, isAccountDoesNotExist(errors.Errorf("Could not do it")))
}

func TestIsDNSChallengeError(t *testing.T) {
	assert.True(t, isDNSChallengeError(&acme.ProblemDetails{
		Type: "urn:ietf:params:acme:error:dns",
	}))

	assert.True(t, isDNSChallengeError(errors.Errorf(
		"acme: error presenting token: DNS problem: SERVFAIL looking up TXT")))

	assert.False(t, isDNSChallengeError(errors.Errorf("Could not do it")))
}

func TestIsTransientError(t *testing.T) {
	assert.True(t, isTransientError(context.DeadlineExceeded))
	assert.True(t, isTransientError(io.ErrUnexpectedEOF))
	assert.True(t, isTransientError(&net.DNSError{IsTimeout: true}))
	assert.True(t, isTransientError(&acme.ProblemDetails{HTTPStatus: 503}))
	assert.True(t, isTransientError(errors.Errorf("dial tcp: connection refused")))

	assert.False(t, isTransientError(errors.Errorf("Could not do it")))
	assert.False(t, isTransientError(&acme.ProblemDetails{
		HTTPStatus: 400,
		Type:       "urn:ietf:params:acme:error:malformed",
	}))
}

func TestGetRetryDelay(t *testing.T) {
	{
		err := &deferredError{
			err:   errors.Errorf("Not ready yet"),
			delay: time.Minute,
		}

		delay := getRetryDelay(1, err)
		assert.True(t, delay >= 48*time.Second && delay <= 72*time.Second, "%s", delay)
	}

	{
		err := &permanentError{err: errors.Errorf("Invalid domain")}

		assert.True(t, getRetryDelay(1, err) >= 10*time.Minute)
		assert.True(t, getRetryDelay(20, err) <= maxRetryDelay)
	}

	{
		err := &acme.RateLimitedError{
			ProblemDetails: &acme.ProblemDetails{Type: acme.RateLimitedErr},
			RetryAfter:     "600",
		}

		assert.Equal(t, 10*time.Minute, getRetryDelay(5, err))
	}

	{
		err := errors.Errorf("connection reset by peer")

		var prev time.Duration
		for i := range 6 {
			delay := getRetryDelay(uint32(i+1), err)
			assert.True(t, delay >= time.Second && delay <= maxRetryDelay, "%s", delay)
			if i > 0 {
				assert.True(t, delay > prev/2, fmt.Sprintf("%s %s", delay, prev))
			}
			prev = delay
		}
	}

	{
		assert.True(t, getRetryDelay(0, errors.Errorf("Could not do it")) >= time.Second)
		assert.True(t, getRetryDelay(1000, errors.Errorf("Could not do it")) <= maxRetryDelay)
	}
}
