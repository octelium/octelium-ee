// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jira

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

const defaultIssueTypeName = "Task"

type Provider struct {
	api           *apiClient
	webhookSecret []byte
	siteURL       string
}

var _ accessintg.Provider = (*Provider)(nil)
var _ accessintg.IdentityResolver = (*Provider)(nil)
var _ accessintg.PresentationDeliverer = (*Provider)(nil)
var _ accessintg.InboundHandler = (*Provider)(nil)
var _ accessintg.DecisionResolver = (*Provider)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface,
	itm *accessv1.Integration) (*Provider, error) {
	spec := itm.Spec.GetJira()
	if spec == nil {
		return nil, errors.Errorf("Not a Jira Integration: %s", itm.Metadata.Name)
	}

	apiToken, err := accessintg.GetSecretValue(ctx, octeliumC, spec.GetApiToken().GetFromSecret())
	if err != nil {
		return nil, err
	}

	ret := &Provider{
		api:     newAPIClient(spec.Url, spec.Email, apiToken),
		siteURL: strings.TrimRight(strings.TrimSpace(spec.Url), "/"),
	}

	if name := spec.GetWebhookSecret().GetFromSecret(); name != "" {
		webhookSecret, err := accessintg.GetSecretValue(ctx, octeliumC, name)
		if err != nil {
			return nil, err
		}
		ret.webhookSecret = []byte(webhookSecret)
	}

	return ret, nil
}

func (p *Provider) Type() accessintg.Type {
	return accessv1.Integration_Status_JIRA
}

func (p *Provider) Capabilities() []accessintg.Capability {
	ret := []accessintg.Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_IDENTITY_RESOLUTION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	}

	if len(p.webhookSecret) > 0 {
		ret = append(ret, accessv1.Integration_Status_INTERACTIVE_REVIEW)
	}

	return ret
}

func Capabilities(spec *accessv1.Integration_Spec_Jira) []accessintg.Capability {
	ret := []accessintg.Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_IDENTITY_RESOLUTION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	}

	if spec.GetWebhookSecret().GetFromSecret() != "" {
		ret = append(ret, accessv1.Integration_Status_INTERACTIVE_REVIEW)
	}

	return ret
}

func (p *Provider) Close() error {
	p.api.hc.CloseIdleConnections()
	return nil
}

func (p *Provider) Validate(ctx context.Context) (*accessintg.TenantInfo, error) {
	usr, err := p.api.myself(ctx)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(p.siteURL)
	if err != nil {
		return nil, err
	}

	return &accessintg.TenantInfo{
		ID:   u.Host,
		Name: usr.DisplayName,
	}, nil
}

func (p *Provider) GetExternalUser(ctx context.Context,
	externalID string) (*accessintg.ExternalUser, error) {
	usr, err := p.api.getUser(ctx, externalID)
	if err != nil {
		return nil, err
	}

	return toExternalUser(usr), nil
}

func (p *Provider) GetExternalUserByEmail(ctx context.Context,
	email string) (*accessintg.ExternalUser, error) {
	usrs, err := p.api.searchUsers(ctx, email)
	if err != nil {
		return nil, err
	}

	var ret *accessintg.ExternalUser

	for _, usr := range usrs {
		if !strings.EqualFold(strings.TrimSpace(usr.EmailAddress), email) {
			continue
		}

		if ret != nil {
			return nil, nil
		}

		ret = toExternalUser(usr)
	}

	return ret, nil
}

func (p *Provider) CreatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	spec, err := targetSpec(in.Target)
	if err != nil {
		return nil, err
	}

	issueTypeName := spec.IssueTypeName
	if issueTypeName == "" {
		issueTypeName = defaultIssueTypeName
	}

	issue, err := p.api.createIssue(ctx, map[string]any{
		"project": map[string]any{
			"key": spec.ProjectKey,
		},
		"issuetype": map[string]any{
			"name": issueTypeName,
		},
		"summary":     buildSummary(in.Presentation),
		"description": buildDescription(in.Presentation),
	})
	if err != nil {
		return nil, err
	}

	if issue == nil || issue.Key == "" {
		return nil, errors.Errorf("Jira did not return an issue key")
	}

	return &accessintg.DeliveryResult{
		ExternalID:          issue.Key,
		ExternalURL:         fmt.Sprintf("%s/browse/%s", p.siteURL, issue.Key),
		ExternalRecipientID: spec.ProjectKey,
	}, nil
}

func (p *Provider) UpdatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	if in.Binding.Status.ExternalID == "" {
		return p.CreatePresentation(ctx, in)
	}

	if err := p.api.updateIssue(ctx, in.Binding.Status.ExternalID, map[string]any{
		"summary":     buildSummary(in.Presentation),
		"description": buildDescription(in.Presentation),
	}); err != nil {
		return nil, err
	}

	return &accessintg.DeliveryResult{
		ExternalID:          in.Binding.Status.ExternalID,
		ExternalURL:         in.Binding.Status.ExternalURL,
		ExternalRecipientID: in.Binding.Status.ExternalRecipientID,
	}, nil
}

func (p *Provider) ClosePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	if in.Binding.Status.ExternalID == "" {
		return &accessintg.DeliveryResult{}, nil
	}

	ret, err := p.UpdatePresentation(ctx, in)
	if err != nil {
		return nil, err
	}

	if err := p.api.addComment(ctx, in.Binding.Status.ExternalID,
		textDocument(fmt.Sprintf(
			"The Octelium access Request %s is now %s and this issue no longer accepts any decision.",
			in.Presentation.RequestName, in.Presentation.State))); err != nil {
		return nil, err
	}

	return ret, nil
}

func (p *Provider) ResolveDecision(ctx context.Context, action *accessintg.Action,
	target *accessv1.IntegrationTarget) (accessv1.Review_Spec_Decision, error) {
	spec, err := targetSpec(target)
	if err != nil {
		return accessv1.Review_Spec_DECISION_UNSET, err
	}

	status := strings.TrimSpace(action.ExternalStatus)
	if status == "" {
		return accessv1.Review_Spec_DECISION_UNSET, nil
	}

	switch {
	case spec.ApproveStatus != "" && strings.EqualFold(status, spec.ApproveStatus):
		return accessv1.Review_Spec_DECISION_APPROVE, nil
	case spec.RejectStatus != "" && strings.EqualFold(status, spec.RejectStatus):
		return accessv1.Review_Spec_DECISION_REJECT, nil
	default:
		return accessv1.Review_Spec_DECISION_UNSET, nil
	}
}

func targetSpec(target *accessv1.IntegrationTarget) (*accessv1.IntegrationTarget_Spec_Jira, error) {
	if target == nil || target.Spec.GetJira() == nil {
		return nil, errors.Errorf("The IntegrationBinding has no Jira IntegrationTarget")
	}

	spec := target.Spec.GetJira()
	if spec.ProjectKey == "" {
		return nil, errors.Errorf("The IntegrationTarget %s has no Jira project key",
			target.Metadata.Name)
	}

	return spec, nil
}

func toExternalUser(usr *jiraUser) *accessintg.ExternalUser {
	if usr == nil || usr.AccountID == "" {
		return nil
	}

	return &accessintg.ExternalUser{
		ID:       usr.AccountID,
		Username: usr.DisplayName,
		Email:    strings.ToLower(strings.TrimSpace(usr.EmailAddress)),
	}
}
