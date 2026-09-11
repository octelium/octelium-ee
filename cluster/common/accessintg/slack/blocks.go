// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package slack

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
)

const (
	actionIDApprove = "octelium-access-approve"
	actionIDReject  = "octelium-access-reject"
)

const maxTextLen = 2800

func buildText(p *accessintg.Presentation) string {
	if p.Purpose == accessv1.IntegrationBinding_Spec_REVIEW_SURFACE.String() {
		return fmt.Sprintf("Access Request %s by %s for %s",
			p.RequestName, p.Requester, p.Resource)
	}

	return fmt.Sprintf("Access Request %s is %s", p.RequestName, p.State)
}

func buildBlocks(p *accessintg.Presentation, target *accessv1.IntegrationTarget,
	bindingName string) []any {
	ret := []any{
		map[string]any{
			"type": "header",
			"text": map[string]any{
				"type": "plain_text",
				"text": truncate(fmt.Sprintf("Access Request %s", p.RequestName), 150),
			},
		},
	}

	fields := []any{
		markdownField("Requester", p.Requester),
		markdownField("Resource", p.Resource),
		markdownField("State", p.State),
	}

	if p.Subject != "" {
		fields = append(fields, markdownField("Subject", p.Subject))
	}
	if p.Urgency != "" {
		fields = append(fields, markdownField("Urgency", p.Urgency))
	}
	if p.RequestedDuration != "" {
		fields = append(fields, markdownField("Requested duration", p.RequestedDuration))
	}
	if p.EffectiveDuration != "" {
		fields = append(fields, markdownField("Effective duration", p.EffectiveDuration))
	}
	if p.PolicyName != "" {
		fields = append(fields, markdownField("Policy", fmt.Sprintf("%s / %s", p.PolicyName, p.RuleName)))
	}
	if p.StepCount > 0 {
		fields = append(fields, markdownField("Review Step",
			fmt.Sprintf("%s (%d of %d)", p.StepName, p.StepIndex, p.StepCount)))
	}
	if p.ApprovalSummary != "" {
		fields = append(fields, markdownField("Quorum", p.ApprovalSummary))
	}

	for _, chunk := range chunkFields(fields, 10) {
		ret = append(ret, map[string]any{
			"type":   "section",
			"fields": chunk,
		})
	}

	if p.Justification != "" {
		ret = append(ret, map[string]any{
			"type": "section",
			"text": mrkdwn(fmt.Sprintf("*Justification*\n%s", escape(p.Justification))),
		})
	}

	if len(p.Decisions) > 0 {
		lines := []string{}
		for _, decision := range p.Decisions {
			line := fmt.Sprintf("*%s* %s", escape(decision.Reviewer), escape(decision.Decision))
			if decision.Justification != "" {
				line = fmt.Sprintf("%s: %s", line, escape(decision.Justification))
			}
			lines = append(lines, line)
		}

		ret = append(ret, map[string]any{
			"type": "section",
			"text": mrkdwn(fmt.Sprintf("*Decisions*\n%s", strings.Join(lines, "\n"))),
		})
	}

	if p.IsActionable {
		ret = append(ret, map[string]any{
			"type": "actions",
			"elements": []any{
				button(actionIDApprove, "Approve", "primary", bindingName),
				button(actionIDReject, "Reject", "danger", bindingName),
			},
		})
	} else if p.Purpose == accessv1.IntegrationBinding_Spec_REVIEW_SURFACE.String() {
		ret = append(ret, map[string]any{
			"type": "context",
			"elements": []any{
				mrkdwn("This Request is no longer awaiting a decision on this Step."),
			},
		})
	}

	contextText := fmt.Sprintf("<%s|Open in the Octelium access portal>", escapeURL(p.PortalURL))
	if mention := targetMention(target); mention != "" {
		contextText = fmt.Sprintf("%s  %s", mention, contextText)
	}

	ret = append(ret, map[string]any{
		"type": "context",
		"elements": []any{
			mrkdwn(contextText),
		},
	})

	return ret
}

func button(actionID, text, style, value string) map[string]any {
	ret := map[string]any{
		"type": "button",
		"text": map[string]any{
			"type": "plain_text",
			"text": text,
		},
		"action_id": actionID,
		"value":     value,
	}

	if style != "" {
		ret["style"] = style
	}

	return ret
}

func markdownField(name, value string) map[string]any {
	return mrkdwn(fmt.Sprintf("*%s*\n%s", name, escape(value)))
}

func mrkdwn(text string) map[string]any {
	return map[string]any{
		"type": "mrkdwn",
		"text": truncate(text, maxTextLen),
	}
}

func chunkFields(fields []any, size int) [][]any {
	ret := [][]any{}

	for len(fields) > size {
		ret = append(ret, fields[:size])
		fields = fields[size:]
	}

	if len(fields) > 0 {
		ret = append(ret, fields)
	}

	return ret
}

func targetMention(target *accessv1.IntegrationTarget) string {
	if target == nil || target.Spec.GetSlack() == nil {
		return ""
	}

	groupID := target.Spec.GetSlack().MentionUserGroupID
	if groupID == "" {
		return ""
	}

	return fmt.Sprintf("<!subteam^%s>", escape(groupID))
}

func escape(arg string) string {
	arg = strings.ReplaceAll(arg, "&", "&amp;")
	arg = strings.ReplaceAll(arg, "<", "&lt;")
	arg = strings.ReplaceAll(arg, ">", "&gt;")

	return arg
}

func escapeURL(arg string) string {
	arg = strings.ReplaceAll(arg, "<", "%3C")
	arg = strings.ReplaceAll(arg, ">", "%3E")
	arg = strings.ReplaceAll(arg, "|", "%7C")

	return arg
}

func truncate(arg string, maxLen int) string {
	if len(arg) <= maxLen {
		return arg
	}

	ret := arg[:maxLen]
	for len(ret) > 0 && !utf8.ValidString(ret) {
		ret = ret[:len(ret)-1]
	}

	return fmt.Sprintf("%s...", ret)
}
