// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accesscmd

import (
	"fmt"
	"unicode/utf8"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

const (
	MaxJustificationLen = 1500
	MaxTicketLen        = 256
	MaxMessageLen       = 500
)

type Problem struct {
	Field   string
	Message string
	Fatal   bool
}

func (p *Problem) Error() string {
	return p.Message
}

func newProblem(field string, format string, args ...any) *Problem {
	return &Problem{
		Field:   field,
		Message: fmt.Sprintf(format, args...),
	}
}

func newFatalProblem(field string, format string, args ...any) *Problem {
	ret := newProblem(field, format, args...)
	ret.Fatal = true
	return ret
}

func ProblemsToError(problems []*Problem) error {
	if len(problems) == 0 {
		return nil
	}

	for _, p := range problems {
		if p.Fatal {
			return grpcutils.InvalidArg("%s", p.Message)
		}
	}

	return grpcutils.InvalidArg("%s", problems[0].Message)
}

func ProblemsToPreview(problems []*Problem) []*accessv1.RequestPreview_Problem {
	ret := make([]*accessv1.RequestPreview_Problem, 0, len(problems))
	for _, p := range problems {
		ret = append(ret, &accessv1.RequestPreview_Problem{
			Field:   p.Field,
			Message: p.Message,
		})
	}

	return ret
}

func HasFatalProblem(problems []*Problem) bool {
	for _, p := range problems {
		if p.Fatal {
			return true
		}
	}

	return false
}

func TextLen(s string) int {
	return utf8.RuneCountInString(s)
}
