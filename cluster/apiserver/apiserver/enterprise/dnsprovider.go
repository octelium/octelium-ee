// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *Server) GetDNSProvider(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.DNSProvider, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetDNSProvider(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListDNSProvider(ctx context.Context, req *enterprisev1.ListDNSProviderOptions) (*enterprisev1.DNSProviderList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListDNSProvider(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) UpdateDNSProvider(ctx context.Context, req *enterprisev1.DNSProvider) (*enterprisev1.DNSProvider, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateDNSProvider(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetDNSProvider(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.EnterpriseC().UpdateDNSProvider(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) validateDNSProvider(ctx context.Context, req *enterprisev1.DNSProvider) error {
	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	switch spec.Type.(type) {
	case *enterprisev1.DNSProvider_Spec_Aws:
		typ := spec.GetAws()
		if typ == nil {
			return grpcutils.InvalidArg("Nil AWS spec")
		}

		if err := s.validateGenStr(typ.AccessKeyID, true, "accessKeyID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.SecretAccessKey); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.Region, true, "region"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.AssumeRoleARN, false, "assumeRoleARN"); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Azure_:
		typ := spec.GetAzure()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Azure spec")
		}

		if err := s.validateGenStr(typ.ClientID, true, "clientID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ClientSecret); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.SubscriptionID, true, "subscriptionID"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.TenantID, true, "tenantID"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ResourceGroupName, true, "resourceGroupName"); err != nil {
			return err
		}
		if err := validateDNSProviderAzureCloud(typ.Cloud); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Cloudflare_:
		typ := spec.GetCloudflare()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Cloudflare spec")
		}

		if !govalidator.IsEmail(typ.Email) {
			return grpcutils.InvalidArg("Invalid email")
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Digitalocean:
		typ := spec.GetDigitalocean()
		if typ == nil {
			return grpcutils.InvalidArg("Nil DigitalOcean spec")
		}

		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Google_:
		typ := spec.GetGoogle()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Google spec")
		}

		if err := s.validateGenStr(typ.Project, true, "project"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ServiceAccount); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Linode_:
		typ := spec.GetLinode()
		if typ == nil {
			return grpcutils.InvalidArg("Nil Linode spec")
		}

		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}

	case *enterprisev1.DNSProvider_Spec_Ovh:
		typ := spec.GetOvh()
		if typ == nil {
			return grpcutils.InvalidArg("Nil OVH spec")
		}

		if err := s.validateGenStr(typ.Endpoint, true, "endpoint"); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ApplicationKey, true, "applicationKey"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApplicationSecret); err != nil {
			return err
		}
		if err := s.validateGenStr(typ.ConsumerKey, true, "consumerKey"); err != nil {
			return err
		}

	default:
		return grpcutils.InvalidArg("You must set DNSProvider type")
	}

	return nil
}

func validateDNSProviderAzureCloud(arg string) error {
	switch arg {
	case "", "public", "china", "usgovernment", "german":
		return nil
	default:
		return grpcutils.InvalidArg("Invalid Azure cloud")
	}
}

type secretOwner interface {
	GetFromSecret() string
}

func (s *Server) validateSecretOwner(ctx context.Context, secOwner secretOwner) error {
	if secOwner == nil {
		return grpcutils.InvalidArg("You must set fromSecret")
	}
	if secOwner.GetFromSecret() == "" {
		return grpcutils.InvalidArg("Empty Secret name")
	}
	if err := apivalidation.ValidateName(secOwner.GetFromSecret(), 0, 0); err != nil {
		return grpcutils.InvalidArg("Invalid Secret name: %s", secOwner.GetFromSecret())
	}

	_, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: secOwner.GetFromSecret()})
	if err == nil {
		return nil
	}
	if !grpcerr.IsNotFound(err) {
		return grpcutils.InternalWithErr(err)
	}
	return grpcutils.InvalidArg("The Secret %s is not found", secOwner.GetFromSecret())
}

func (s *Server) validateGenStr(arg string, required bool, name string) error {
	if arg == "" {
		if required {
			return grpcutils.InvalidArg("%s is required", name)
		}
		return nil
	}

	if len(arg) > 256 {
		return grpcutils.InvalidArg("%s is too long", name)
	}
	if !govalidator.IsASCII(arg) {
		return grpcutils.InvalidArg("%s is invalid", name)
	}

	return nil
}
