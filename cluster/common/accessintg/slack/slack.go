// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package slack

import (
	"context"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

type Provider struct {
	api           *apiClient
	signingSecret []byte
	teamID        string
}

var _ accessintg.Provider = (*Provider)(nil)
var _ accessintg.IdentityResolver = (*Provider)(nil)
var _ accessintg.PresentationDeliverer = (*Provider)(nil)
var _ accessintg.InboundHandler = (*Provider)(nil)

func New(ctx context.Context, octeliumC octeliumc.ClientInterface,
	itm *accessv1.Integration) (*Provider, error) {
	spec := itm.Spec.GetSlack()
	if spec == nil {
		return nil, errors.Errorf("Not a Slack Integration: %s", itm.Metadata.Name)
	}

	botToken, err := accessintg.GetSecretValue(ctx, octeliumC, spec.GetBotToken().GetFromSecret())
	if err != nil {
		return nil, err
	}

	signingSecret, err := accessintg.GetSecretValue(ctx, octeliumC,
		spec.GetSigningSecret().GetFromSecret())
	if err != nil {
		return nil, err
	}

	teamID := spec.TeamID
	if teamID == "" && itm.Status != nil {
		teamID = itm.Status.ExternalTenantID
	}

	return &Provider{
		api:           newAPIClient(spec.BaseURL, botToken),
		signingSecret: []byte(signingSecret),
		teamID:        teamID,
	}, nil
}

func (p *Provider) Type() accessintg.Type {
	return accessv1.Integration_Status_SLACK
}

func (p *Provider) Capabilities() []accessintg.Capability {
	return Capabilities(nil)
}

func Capabilities(spec *accessv1.Integration_Spec_Slack) []accessintg.Capability {
	return []accessintg.Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY,
		accessv1.Integration_Status_INTERACTIVE_REVIEW,
		accessv1.Integration_Status_REQUEST_CREATION,
		accessv1.Integration_Status_IDENTITY_RESOLUTION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	}
}

func (p *Provider) Close() error {
	p.api.hc.CloseIdleConnections()
	return nil
}

func (p *Provider) Validate(ctx context.Context) (*accessintg.TenantInfo, error) {
	resp, err := p.api.authTest(ctx)
	if err != nil {
		return nil, err
	}

	if p.teamID != "" && resp.TeamID != p.teamID {
		return nil, errors.Errorf(
			"The Slack bot token belongs to the workspace %s instead of %s",
			resp.TeamID, p.teamID)
	}

	return &accessintg.TenantInfo{
		ID:   resp.TeamID,
		Name: resp.Team,
	}, nil
}

func (p *Provider) GetExternalUser(ctx context.Context,
	externalID string) (*accessintg.ExternalUser, error) {
	usr, err := p.api.usersInfo(ctx, externalID)
	if err != nil {
		return nil, err
	}

	return toExternalUser(usr), nil
}

func (p *Provider) GetExternalUserByEmail(ctx context.Context,
	email string) (*accessintg.ExternalUser, error) {
	usr, err := p.api.usersLookupByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return toExternalUser(usr), nil
}

func (p *Provider) CreatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	channel, err := p.resolveChannel(ctx, in)
	if err != nil {
		return nil, err
	}

	resp, err := p.api.chatPostMessage(ctx, channel,
		buildText(in.Presentation),
		buildBlocks(in.Presentation, in.Target, in.Binding.Metadata.Name))
	if err != nil {
		return nil, err
	}

	if resp.TS == "" {
		return nil, errors.Errorf("Slack did not return a message timestamp")
	}

	recipientID := resp.Channel
	if recipientID == "" {
		recipientID = channel
	}

	return &accessintg.DeliveryResult{
		ExternalID:          resp.TS,
		ExternalURL:         p.api.chatGetPermalink(ctx, recipientID, resp.TS),
		ExternalRecipientID: recipientID,
	}, nil
}

func (p *Provider) UpdatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	if in.Binding.Status.ExternalID == "" || in.Binding.Status.ExternalRecipientID == "" {
		return p.CreatePresentation(ctx, in)
	}

	if _, err := p.api.chatUpdate(ctx,
		in.Binding.Status.ExternalRecipientID, in.Binding.Status.ExternalID,
		buildText(in.Presentation),
		buildBlocks(in.Presentation, in.Target, in.Binding.Metadata.Name)); err != nil {
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
	if in.Binding.Status.ExternalID == "" || in.Binding.Status.ExternalRecipientID == "" {
		return &accessintg.DeliveryResult{}, nil
	}

	return p.UpdatePresentation(ctx, in)
}

func (p *Provider) resolveChannel(ctx context.Context,
	in *accessintg.PresentationDelivery) (string, error) {
	if in.Binding.Status.ExternalRecipientID != "" {
		return in.Binding.Status.ExternalRecipientID, nil
	}

	if in.Target != nil {
		slack := in.Target.Spec.GetSlack()
		if slack == nil || slack.ChannelID == "" {
			return "", errors.Errorf("The IntegrationTarget %s has no Slack channel",
				in.Target.Metadata.Name)
		}

		return slack.ChannelID, nil
	}

	if in.RecipientID == "" {
		return "", errors.Errorf("The IntegrationBinding %s has no Slack recipient",
			in.Binding.Metadata.Name)
	}

	return p.api.conversationsOpen(ctx, in.RecipientID)
}

func toExternalUser(usr *user) *accessintg.ExternalUser {
	if usr == nil || usr.ID == "" || usr.Deleted {
		return nil
	}

	ret := &accessintg.ExternalUser{
		ID:       usr.ID,
		Username: usr.Name,
	}

	if usr.Profile != nil {
		ret.Email = strings.ToLower(strings.TrimSpace(usr.Profile.Email))
		if ret.Username == "" {
			ret.Username = usr.Profile.RealName
		}
	}

	return ret
}
