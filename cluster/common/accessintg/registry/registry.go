// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package registry

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/jira"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/slack"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/webhook"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface,
	itm *accessv1.Integration) (accessintg.Provider, error) {
	if itm == nil || itm.Spec == nil {
		return nil, errors.Errorf("Nil Integration")
	}

	switch itm.Spec.Type.(type) {
	case *accessv1.Integration_Spec_Slack_:
		return slack.New(ctx, octeliumC, itm)
	case *accessv1.Integration_Spec_Jira_:
		return jira.New(ctx, octeliumC, itm)
	case *accessv1.Integration_Spec_Webhook_:
		return webhook.New(ctx, octeliumC, itm)
	default:
		return nil, errors.Errorf("The Integration %s has no type", itm.Metadata.Name)
	}
}

func TypeOf(itm *accessv1.Integration) accessintg.Type {
	if itm == nil || itm.Spec == nil {
		return accessv1.Integration_Status_TYPE_UNSET
	}

	switch itm.Spec.Type.(type) {
	case *accessv1.Integration_Spec_Slack_:
		return accessv1.Integration_Status_SLACK
	case *accessv1.Integration_Spec_Jira_:
		return accessv1.Integration_Status_JIRA
	case *accessv1.Integration_Spec_Webhook_:
		return accessv1.Integration_Status_WEBHOOK
	default:
		return accessv1.Integration_Status_TYPE_UNSET
	}
}

func CapabilitiesOf(itm *accessv1.Integration) []accessintg.Capability {
	if itm == nil || itm.Spec == nil {
		return nil
	}

	switch itm.Spec.Type.(type) {
	case *accessv1.Integration_Spec_Slack_:
		return slack.Capabilities(itm.Spec.GetSlack())
	case *accessv1.Integration_Spec_Jira_:
		return jira.Capabilities(itm.Spec.GetJira())
	case *accessv1.Integration_Spec_Webhook_:
		return webhook.Capabilities(itm.Spec.GetWebhook())
	default:
		return nil
	}
}
