// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/pkg/errors"
)

type Sink struct {
	h *H

	GRPCEndpoint string
	HTTPEndpoint string

	pod       string
	namespace string
	dir       string
}

func (h *H) Sink(t *testing.T) *Sink {
	t.Helper()

	return &Sink{
		h:            h,
		GRPCEndpoint: h.StateValue(t, eescenario.StateSinkGRPC),
		HTTPEndpoint: h.StateValue(t, eescenario.StateSinkHTTP),
		pod:          h.StateValue(t, eescenario.StateSinkPod),
		namespace:    h.StateValue(t, eescenario.StateSinkNS),
		dir:          h.StateValue(t, eescenario.StateSinkDir),
	}
}

func (s *Sink) read(ctx context.Context, file string) (string, error) {
	out, err := s.h.Output(ctx, fmt.Sprintf(
		`kubectl exec -n %s %s -- cat %s/%s 2>/dev/null || true`,
		s.namespace, s.pod, s.dir, file))
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (s *Sink) Logs(t *testing.T) string {
	t.Helper()
	return s.mustRead(t, "logs.json")
}

func (s *Sink) Metrics(t *testing.T) string {
	t.Helper()
	return s.mustRead(t, "metrics.json")
}

func (s *Sink) mustRead(t *testing.T, file string) string {
	t.Helper()

	ctx, cancel := s.h.Ctx(t)
	defer cancel()

	out, err := s.read(ctx, file)
	if err != nil {
		t.Fatalf("Could not read %s from the OTLP sink: %+v", file, err)
	}

	return out
}

func (s *Sink) Truncate(t *testing.T) {
	t.Helper()

	s.h.MustRun(t, fmt.Sprintf(
		`kubectl exec -n %s %s -- sh -c 'rm -f %s/logs.json %s/metrics.json'`,
		s.namespace, s.pod, s.dir, s.dir))
}

func (s *Sink) WaitContains(t *testing.T, file, needle string, budget time.Duration) {
	t.Helper()

	s.h.Eventually(t,
		fmt.Sprintf("the OTLP sink %s to contain %q", file, needle), budget,
		func(ctx context.Context) error {
			out, err := s.read(ctx, file)
			if err != nil {
				return err
			}
			if !strings.Contains(out, needle) {
				return errors.Errorf("the sink %s has %d bytes and none match", file, len(out))
			}
			return nil
		})
}

func (s *Sink) WaitLogs(t *testing.T, needle string, budget time.Duration) {
	t.Helper()
	s.WaitContains(t, "logs.json", needle, budget)
}

func (s *Sink) WaitMetrics(t *testing.T, needle string, budget time.Duration) {
	t.Helper()
	s.WaitContains(t, "metrics.json", needle, budget)
}

func (s *Sink) MustStayEmpty(t *testing.T, file string, window time.Duration) {
	t.Helper()

	s.h.Consistently(t, fmt.Sprintf("the OTLP sink %s to stay empty", file), window,
		func(ctx context.Context) error {
			out, err := s.read(ctx, file)
			if err != nil {
				return nil
			}
			if strings.TrimSpace(out) != "" {
				return errors.Errorf("the sink %s received %d bytes", file, len(out))
			}
			return nil
		})
}
