// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jira

import (
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
)

const maxSummaryLen = 250

func buildSummary(p *accessintg.Presentation) string {
	summary := fmt.Sprintf("Octelium access Request %s by %s for %s",
		p.RequestName, p.Requester, p.Resource)

	if len(summary) > maxSummaryLen {
		summary = summary[:maxSummaryLen]
	}

	return summary
}

func buildDescription(p *accessintg.Presentation) map[string]any {
	content := []any{
		paragraph(fmt.Sprintf("The Octelium access Request %s is currently %s.",
			p.RequestName, p.State)),
	}

	rows := [][2]string{
		{"Requester", p.Requester},
		{"Subject", p.Subject},
		{"Resource", p.Resource},
		{"Urgency", p.Urgency},
		{"Requested duration", p.RequestedDuration},
		{"Effective duration", p.EffectiveDuration},
		{"Policy", p.PolicyName},
		{"Rule", p.RuleName},
		{"Review Step", stepLabel(p)},
		{"Quorum", p.ApprovalSummary},
		{"Justification", p.Justification},
	}

	items := []any{}
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		items = append(items, listItem(fmt.Sprintf("%s: %s", row[0], row[1])))
	}

	if len(items) > 0 {
		content = append(content, map[string]any{
			"type":    "bulletList",
			"content": items,
		})
	}

	if len(p.Decisions) > 0 {
		decisions := []any{}
		for _, decision := range p.Decisions {
			line := fmt.Sprintf("%s: %s", decision.Reviewer, decision.Decision)
			if decision.Justification != "" {
				line = fmt.Sprintf("%s (%s)", line, decision.Justification)
			}
			decisions = append(decisions, listItem(line))
		}

		content = append(content,
			paragraph("Decisions"),
			map[string]any{
				"type":    "bulletList",
				"content": decisions,
			})
	}

	content = append(content, paragraph(fmt.Sprintf(
		"The Octelium access portal is the authoritative record of this Request: %s",
		p.PortalURL)))

	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

func textDocument(text string) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			paragraph(text),
		},
	}
}

func paragraph(text string) map[string]any {
	return map[string]any{
		"type": "paragraph",
		"content": []any{
			map[string]any{
				"type": "text",
				"text": text,
			},
		},
	}
}

func listItem(text string) map[string]any {
	return map[string]any{
		"type": "listItem",
		"content": []any{
			paragraph(text),
		},
	}
}

func stepLabel(p *accessintg.Presentation) string {
	if p.StepCount == 0 {
		return ""
	}

	return fmt.Sprintf("%s (%d of %d)", p.StepName, p.StepIndex, p.StepCount)
}
