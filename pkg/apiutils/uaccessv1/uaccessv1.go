// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package uaccessv1

import (
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	KindCatalog = "Catalog"
	KindPolicy  = "Policy"
	KindRequest = "Request"
	KindReview  = "Review"
	KindSecret  = "Secret"

	KindIntegration         = "Integration"
	KindIntegrationIdentity = "IntegrationIdentity"
	KindIntegrationBinding  = "IntegrationBinding"
)

type ResourceObjectRefG interface {
	*accessv1.Catalog | *accessv1.Policy | *accessv1.Request | *accessv1.Review |
		*accessv1.Secret | *accessv1.Integration |
		*accessv1.IntegrationIdentity | *accessv1.IntegrationBinding
}

const API = "access"
const Version = "v1"
const APIVersion = "access/v1"

type Catalog struct {
	*accessv1.Catalog
}

type CatalogList struct {
	*accessv1.CatalogList
}

type Policy struct {
	*accessv1.Policy
}

type PolicyList struct {
	*accessv1.PolicyList
}

type Request struct {
	*accessv1.Request
}

type RequestList struct {
	*accessv1.RequestList
}

type Review struct {
	*accessv1.Review
}

type ReviewList struct {
	*accessv1.ReviewList
}

type Secret struct {
	*accessv1.Secret
}

type SecretList struct {
	*accessv1.SecretList
}

type Integration struct {
	*accessv1.Integration
}

type IntegrationList struct {
	*accessv1.IntegrationList
}

type IntegrationIdentity struct {
	*accessv1.IntegrationIdentity
}

type IntegrationIdentityList struct {
	*accessv1.IntegrationIdentityList
}

type IntegrationBinding struct {
	*accessv1.IntegrationBinding
}

type IntegrationBindingList struct {
	*accessv1.IntegrationBindingList
}

func NewObjectList(kind string) (umetav1.ObjectI, error) {

	switch kind {
	case KindCatalog:
		return &accessv1.CatalogList{}, nil
	case KindPolicy:
		return &accessv1.PolicyList{}, nil
	case KindRequest:
		return &accessv1.RequestList{}, nil
	case KindReview:
		return &accessv1.ReviewList{}, nil
	case KindSecret:
		return &accessv1.SecretList{}, nil
	case KindIntegration:
		return &accessv1.IntegrationList{}, nil
	case KindIntegrationIdentity:
		return &accessv1.IntegrationIdentityList{}, nil
	case KindIntegrationBinding:
		return &accessv1.IntegrationBindingList{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func NewObjectListOptions(kind string) (proto.Message, error) {

	switch kind {
	case KindCatalog:
		return &accessv1.ListCatalogOptions{}, nil
	case KindPolicy:
		return &accessv1.ListPolicyOptions{}, nil
	case KindRequest:
		return &accessv1.ListRequestOptions{}, nil
	case KindReview:
		return &accessv1.ListReviewOptions{}, nil
	case KindSecret:
		return &accessv1.ListSecretOptions{}, nil
	case KindIntegration:
		return &accessv1.ListIntegrationOptions{}, nil
	case KindIntegrationIdentity:
		return &accessv1.ListIntegrationIdentityOptions{}, nil
	case KindIntegrationBinding:
		return &accessv1.ListIntegrationBindingOptions{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func NewObject(kind string) (umetav1.ResourceObjectI, error) {

	switch kind {
	case KindCatalog:
		return &accessv1.Catalog{}, nil
	case KindPolicy:
		return &accessv1.Policy{}, nil
	case KindRequest:
		return &accessv1.Request{}, nil
	case KindReview:
		return &accessv1.Review{}, nil
	case KindSecret:
		return &accessv1.Secret{}, nil
	case KindIntegration:
		return &accessv1.Integration{}, nil
	case KindIntegrationIdentity:
		return &accessv1.IntegrationIdentity{}, nil
	case KindIntegrationBinding:
		return &accessv1.IntegrationBinding{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func ToCatalog(a *accessv1.Catalog) *Catalog {
	return &Catalog{
		Catalog: a,
	}
}

func ToCatalogList(a *accessv1.CatalogList) *CatalogList {
	return &CatalogList{
		CatalogList: a,
	}
}

func ToPolicy(a *accessv1.Policy) *Policy {
	return &Policy{
		Policy: a,
	}
}

func ToPolicyList(a *accessv1.PolicyList) *PolicyList {
	return &PolicyList{
		PolicyList: a,
	}
}

func ToRequest(a *accessv1.Request) *Request {
	return &Request{
		Request: a,
	}
}

func ToRequestList(a *accessv1.RequestList) *RequestList {
	return &RequestList{
		RequestList: a,
	}
}

func ToReview(a *accessv1.Review) *Review {
	return &Review{
		Review: a,
	}
}

func ToReviewList(a *accessv1.ReviewList) *ReviewList {
	return &ReviewList{
		ReviewList: a,
	}
}

func ToSecret(a *accessv1.Secret) *Secret {
	return &Secret{
		Secret: a,
	}
}

func ToSecretList(a *accessv1.SecretList) *SecretList {
	return &SecretList{
		SecretList: a,
	}
}

func ToIntegration(a *accessv1.Integration) *Integration {
	return &Integration{
		Integration: a,
	}
}

func ToIntegrationList(a *accessv1.IntegrationList) *IntegrationList {
	return &IntegrationList{
		IntegrationList: a,
	}
}

func ToIntegrationIdentity(a *accessv1.IntegrationIdentity) *IntegrationIdentity {
	return &IntegrationIdentity{
		IntegrationIdentity: a,
	}
}

func ToIntegrationIdentityList(a *accessv1.IntegrationIdentityList) *IntegrationIdentityList {
	return &IntegrationIdentityList{
		IntegrationIdentityList: a,
	}
}

func ToIntegrationBinding(a *accessv1.IntegrationBinding) *IntegrationBinding {
	return &IntegrationBinding{
		IntegrationBinding: a,
	}
}

func ToIntegrationBindingList(a *accessv1.IntegrationBindingList) *IntegrationBindingList {
	return &IntegrationBindingList{
		IntegrationBindingList: a,
	}
}

func (s *Secret) GetValueStr() string {
	if s.Data == nil {
		return ""
	}
	switch s.Data.Type.(type) {
	case *accessv1.Secret_Data_Value:
		return s.Data.GetValue()
	case *accessv1.Secret_Data_ValueBytes:
		return string(s.Data.GetValueBytes())
	default:
		return ""
	}
}

func (s *Secret) GetValueBytes() []byte {
	return []byte(s.GetValueStr())
}

func (s *SecretList) GetByName(name string) (*accessv1.Secret, error) {
	for _, itm := range s.Items {
		if itm.Metadata.Name == name {
			return itm, nil
		}
	}
	return nil, errors.Errorf("No Secret exists with name: %s", name)
}
