// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

const maxRespBytes = 1 << 20

type Provider struct {
	url           string
	signingSecret []byte
	inboundSecret []byte
	hc            *http.Client
}

var _ accessintg.Provider = (*Provider)(nil)
var _ accessintg.PresentationDeliverer = (*Provider)(nil)
var _ accessintg.InboundHandler = (*Provider)(nil)

type deliveryPayload struct {
	Type         string                   `json:"type"`
	BindingName  string                   `json:"bindingName"`
	TargetName   string                   `json:"targetName,omitempty"`
	ExternalID   string                   `json:"externalID,omitempty"`
	Presentation *accessintg.Presentation `json:"presentation"`
	SentAt       string                   `json:"sentAt"`
}

func New(ctx context.Context, octeliumC octeliumc.ClientInterface,
	itm *accessv1.Integration) (*Provider, error) {
	spec := itm.Spec.GetWebhook()
	if spec == nil {
		return nil, errors.Errorf("Not a Webhook Integration: %s", itm.Metadata.Name)
	}

	signingSecret, err := accessintg.GetSecretValue(ctx, octeliumC,
		spec.GetSigningSecret().GetFromSecret())
	if err != nil {
		return nil, err
	}

	ret := &Provider{
		url:           strings.TrimSpace(spec.Url),
		signingSecret: []byte(signingSecret),
		hc: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	if name := spec.GetInboundSecret().GetFromSecret(); name != "" {
		inboundSecret, err := accessintg.GetSecretValue(ctx, octeliumC, name)
		if err != nil {
			return nil, err
		}
		ret.inboundSecret = []byte(inboundSecret)
	}

	return ret, nil
}

func (p *Provider) Type() accessintg.Type {
	return accessv1.Integration_Status_WEBHOOK
}

func (p *Provider) Capabilities() []accessintg.Capability {
	ret := []accessintg.Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	}

	if len(p.inboundSecret) > 0 {
		ret = append(ret,
			accessv1.Integration_Status_INTERACTIVE_REVIEW,
			accessv1.Integration_Status_REQUEST_CREATION)
	}

	return ret
}

func Capabilities(spec *accessv1.Integration_Spec_Webhook) []accessintg.Capability {
	ret := []accessintg.Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	}

	if spec.GetInboundSecret().GetFromSecret() != "" {
		ret = append(ret,
			accessv1.Integration_Status_INTERACTIVE_REVIEW,
			accessv1.Integration_Status_REQUEST_CREATION)
	}

	return ret
}

func (p *Provider) Close() error {
	p.hc.CloseIdleConnections()
	return nil
}

func (p *Provider) Validate(ctx context.Context) (*accessintg.TenantInfo, error) {
	u, err := url.Parse(p.url)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, errors.Errorf("The webhook URL scheme must be http or https")
	}

	if u.Host == "" {
		return nil, errors.Errorf("The webhook URL has no host")
	}

	return &accessintg.TenantInfo{
		ID:   u.Host,
		Name: u.Host,
	}, nil
}

func (p *Provider) CreatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	return p.deliver(ctx, "created", in)
}

func (p *Provider) UpdatePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	return p.deliver(ctx, "updated", in)
}

func (p *Provider) ClosePresentation(ctx context.Context,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	return p.deliver(ctx, "closed", in)
}

func (p *Provider) deliver(ctx context.Context, typ string,
	in *accessintg.PresentationDelivery) (*accessintg.DeliveryResult, error) {
	u := p.url
	targetName := ""

	if in.Target != nil {
		targetName = in.Target.Metadata.Name
		if spec := in.Target.Spec.GetWebhook(); spec != nil {
			if spec.Url != "" {
				u = spec.Url
			}
			if spec.Name != "" {
				targetName = spec.Name
			}
		}
	}

	payload := &deliveryPayload{
		Type:         typ,
		BindingName:  in.Binding.Metadata.Name,
		TargetName:   targetName,
		ExternalID:   in.Binding.Status.ExternalID,
		Presentation: in.Presentation,
		SentAt:       time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Octelium-Timestamp", timestamp)
	req.Header.Set("X-Octelium-Signature", sign(p.signingSecret, timestamp, body))

	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxRespBytes)); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.Errorf("The webhook returned the status code %d", resp.StatusCode)
	}

	externalID := in.Binding.Status.ExternalID
	if externalID == "" {
		externalID = in.Binding.Metadata.Name
	}

	return &accessintg.DeliveryResult{
		ExternalID:          externalID,
		ExternalURL:         in.Binding.Status.ExternalURL,
		ExternalRecipientID: targetName,
	}, nil
}

func sign(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)

	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}
