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
	stderrors "errors"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/acme"
)

var retryAfterRegexp = regexp.MustCompile(
	`(?i)retry after[:\s]+(` +
		`[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?Z` +
		`|[0-9]{4}-[0-9]{2}-[0-9]{2}\s+[0-9]{2}:[0-9]{2}:[0-9]{2}\s+(?:UTC|GMT)` +
		`)`)

type deferredError struct {
	err   error
	delay time.Duration
}

func (e *deferredError) Error() string {
	return e.err.Error()
}

func (e *deferredError) Unwrap() error {
	return e.err
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string {
	return e.err.Error()
}

func (e *permanentError) Unwrap() error {
	return e.err
}

func getRetryDelay(failures uint32, err error) time.Duration {
	if failures < 1 {
		failures = 1
	}

	if delay := getRetryAfter(err); delay > 0 {
		return clampRetryDelay(delay)
	}

	base := minRetryDelay
	max := 30 * time.Minute

	var deferred *deferredError
	var permanent *permanentError

	switch {
	case stderrors.As(err, &deferred):
		base = deferred.delay
		max = 15 * time.Minute
	case stderrors.As(err, &permanent):
		base = 15 * time.Minute
		max = maxRetryDelay
	case isBadNonce(err):
		base = 5 * time.Second
		max = 2 * time.Minute
	case isRateLimited(err):
		base = time.Hour
		max = maxRetryDelay
	case isDNSChallengeError(err):
		base = 2 * time.Minute
		max = time.Hour
	case !isTransientError(err):
		base = 5 * time.Minute
		max = maxRetryDelay
	}

	if base <= 0 {
		base = minRetryDelay
	}

	delay := time.Duration(float64(base) *
		math.Pow(2, math.Min(float64(failures-1), 10)))
	if delay > max || delay <= 0 {
		delay = max
	}

	return clampRetryDelay(addJitter(delay))
}

func clampRetryDelay(delay time.Duration) time.Duration {
	if delay < time.Second {
		return time.Second
	}
	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}

func addJitter(delay time.Duration) time.Duration {
	return time.Duration(float64(delay) * (0.8 + rand.Float64()*0.4))
}

func getRetryAfter(err error) time.Duration {
	var rateLimited *acme.RateLimitedError
	if stderrors.As(err, &rateLimited) {
		if delay := parseRetryAfter(rateLimited.RetryAfter); delay > 0 {
			return delay
		}
	}

	if matches := retryAfterRegexp.FindStringSubmatch(err.Error()); len(matches) == 2 {
		return parseRetryAfter(matches[1])
	}

	return 0
}

func parseRetryAfter(val string) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(val, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	if at, err := http.ParseTime(val); err == nil {
		return time.Until(at)
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05 MST",
	} {
		if at, err := time.Parse(layout, val); err == nil {
			return time.Until(at)
		}
	}

	return 0
}

func isBadNonce(err error) bool {
	var nonceErr *acme.NonceError
	if stderrors.As(err, &nonceErr) {
		return true
	}

	var problem *acme.ProblemDetails
	return stderrors.As(err, &problem) && problem.Type == acme.BadNonceErr
}

func isRateLimited(err error) bool {
	var rateLimited *acme.RateLimitedError
	if stderrors.As(err, &rateLimited) {
		return true
	}

	var problem *acme.ProblemDetails
	if stderrors.As(err, &problem) {
		return problem.Type == acme.RateLimitedErr ||
			problem.HTTPStatus == http.StatusTooManyRequests
	}

	val := strings.ToLower(err.Error())
	return strings.Contains(val, "rate limit") ||
		strings.Contains(val, "too many requests") ||
		strings.Contains(val, "status code: 429")
}

func isDNSChallengeError(err error) bool {
	var problem *acme.ProblemDetails
	if stderrors.As(err, &problem) {
		switch {
		case strings.HasSuffix(problem.Type, ":dns"),
			strings.HasSuffix(problem.Type, ":connection"),
			strings.HasSuffix(problem.Type, ":unauthorized"):
			return true
		}
	}

	val := strings.ToLower(err.Error())
	return strings.Contains(val, "dns problem") ||
		strings.Contains(val, "propagation") ||
		strings.Contains(val, "servfail") ||
		strings.Contains(val, "could not find zone") ||
		(strings.Contains(val, "challenge") && strings.Contains(val, "timeout"))
}

func isAccountDoesNotExist(err error) bool {
	var problem *acme.ProblemDetails
	if stderrors.As(err, &problem) {
		return strings.HasSuffix(problem.Type, ":accountDoesNotExist")
	}

	val := strings.ToLower(err.Error())
	return strings.Contains(val, "accountdoesnotexist") ||
		strings.Contains(val, "account does not exist")
}

func isTransientError(err error) bool {
	if stderrors.Is(err, context.DeadlineExceeded) ||
		stderrors.Is(err, io.EOF) ||
		stderrors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if stderrors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var problem *acme.ProblemDetails
	if stderrors.As(err, &problem) {
		switch {
		case problem.HTTPStatus >= 500,
			problem.HTTPStatus == http.StatusRequestTimeout,
			problem.HTTPStatus == http.StatusTooManyRequests,
			problem.Type == acme.BadNonceErr,
			strings.HasSuffix(problem.Type, ":serverInternal"),
			strings.HasSuffix(problem.Type, ":connection"),
			strings.HasSuffix(problem.Type, ":dns"):
			return true
		}
		return false
	}

	val := strings.ToLower(err.Error())
	for _, itm := range []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"temporary failure",
		"temporarily unavailable",
		"service unavailable",
		"gateway timeout",
		"bad gateway",
		"internal server error",
		"unexpected eof",
		"i/o timeout",
		"tls handshake timeout",
		"no such host",
		"server misbehaving",
		"status code: 408",
		"status code: 500",
		"status code: 502",
		"status code: 503",
		"status code: 504",
	} {
		if strings.Contains(val, itm) {
			return true
		}
	}

	return false
}
