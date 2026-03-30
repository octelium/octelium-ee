// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package awskms

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/identity"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type store struct {
	c     *kms.Client
	keyID string
	store *enterprisev1.SecretStore
}

type credI struct {
	assumeRoleResp *sts.AssumeRoleWithWebIdentityOutput
}

func (t *credI) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     *t.assumeRoleResp.Credentials.AccessKeyId,
		SecretAccessKey: *t.assumeRoleResp.Credentials.SecretAccessKey,
		SessionToken:    *t.assumeRoleResp.Credentials.SessionToken,
		CanExpire:       true,
		Expires:         *t.assumeRoleResp.Credentials.Expiration,
	}, nil
}

func NewStore(ctx context.Context, opts *stores.StoreOpts) (*store, error) {
	ret := &store{
		store: opts.SecretStore,
	}

	spec := opts.SecretStore.Spec.GetAwsKeyManagementService()
	if spec == nil {
		return nil, errors.Errorf("Not a AWS Secret Manager")
	}

	/*
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(spec.Region),
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     spec.AccessKeyID,
					SecretAccessKey: "",
				},
			}))
		if err != nil {
			return nil, err
		}
	*/

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	stsClient := sts.NewFromConfig(cfg)
	tkn, err := identity.GetIdentityToken(ctx, opts.SecretStore)
	if err != nil {
		return nil, err
	}

	assumeRoleResp, err := stsClient.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &spec.RoleARN,
		RoleSessionName:  aws.String(fmt.Sprintf("octelium-region-%s", vutils.GetMyRegionName())),
		WebIdentityToken: &tkn.Token,
	})
	if err != nil {
		return nil, err
	}

	ret.c = kms.New(kms.Options{
		Credentials: &credI{
			assumeRoleResp: assumeRoleResp,
		},
		Region: spec.Region,
	})
	ret.keyID = spec.KeyID

	return ret, nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {

	resp, err := s.c.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     utils_types.StrToPtr(s.keyID),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return resp.CiphertextBlob, nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {

	res, err := s.c.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          utils_types.StrToPtr(s.keyID),
		CiphertextBlob: ciphertext,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return res.Plaintext, nil
}

func (s *store) UID() string {
	return s.store.Metadata.Uid
}

func (s *store) Close() error {

	return nil
}

func (s *store) Initialize(ctx context.Context) error {
	return nil
}
