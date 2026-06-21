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
)

type ResourceObjectRefG interface {
	*accessv1.Catalog | *accessv1.Policy | *accessv1.Request | *accessv1.Review
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
