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

/*

func (s *Server) CreateDNSProvider(ctx context.Context, req *enterprisev1.DNSProvider) (*enterprisev1.DNSProvider, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	{
		_, err := s.octeliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
		if err == nil {
			return nil, grpcutils.AlreadyExists("The DNSProvider %s already exists", req.Metadata.Name)
		}
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	if err := s.validateDNSProvider(ctx, req); err != nil {
		return nil, err
	}

	item := &enterprisev1.DNSProvider{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.DNSProvider_Status{},
	}

	item, err := s.octeliumC.EnterpriseC().CreateDNSProvider(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

*/

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

/*
func (s *Server) DeleteDNSProvider(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	if g.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot delete the system group: %s", req.Name)
	}

	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	_, err = s.octeliumC.EnterpriseC().DeleteDNSProvider(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

*/

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
		return nil, err
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
	/*
		case *enterprisev1.DNSProvider_Spec_Alibaba_:
			typ := spec.GetAlibaba()
			if err := s.validateGenStr(typ.AccessKeyID, true, "accessKeyID"); err != nil {
				return err
			}
			if err := s.validateSecretOwner(ctx, typ.AccessKeySecret); err != nil {
				return err
			}
	*/
	case *enterprisev1.DNSProvider_Spec_Aws:
		typ := spec.GetAws()
		if err := s.validateGenStr(typ.AccessKeyID, true, "accessKeyID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.SecretAccessKey); err != nil {
			return err
		}
	case *enterprisev1.DNSProvider_Spec_Azure_:
		typ := spec.GetAzure()
		if err := s.validateGenStr(typ.ClientID, true, "clientID"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ClientSecret); err != nil {
			return err
		}
	case *enterprisev1.DNSProvider_Spec_Cloudflare_:
		typ := spec.GetCloudflare()
		if !govalidator.IsEmail(typ.Email) {
			return grpcutils.InvalidArg("Invalid email")
		}
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}
	case *enterprisev1.DNSProvider_Spec_Digitalocean:
		typ := spec.GetDigitalocean()
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}
	case *enterprisev1.DNSProvider_Spec_Google_:
		typ := spec.GetGoogle()
		if err := s.validateGenStr(typ.Project, true, "project"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ServiceAccount); err != nil {
			return err
		}

		/*
			case *enterprisev1.DNSProvider_Spec_Hetzner_:
				typ := spec.GetHetzner()
				if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
					return err
				}
		*/
	case *enterprisev1.DNSProvider_Spec_Linode_:
		typ := spec.GetLinode()
		if err := s.validateSecretOwner(ctx, typ.ApiToken); err != nil {
			return err
		}
	case *enterprisev1.DNSProvider_Spec_Ovh:
		typ := spec.GetOvh()
		if err := s.validateGenStr(typ.ApplicationKey, true, "applicationKey"); err != nil {
			return err
		}
		if err := s.validateSecretOwner(ctx, typ.ApplicationSecret); err != nil {
			return err
		}

		/*
			case *enterprisev1.DNSProvider_Spec_Smtp:
				typ := spec.GetSmtp()
				if err := s.validateGenStr(typ.Username, true, "username"); err != nil {
					return err
				}
				if !govalidator.IsHost(typ.Host) {
					return grpcutils.InvalidArg("Invalid host")
				}
				if err := s.validateSecretOwner(ctx, typ.Password); err != nil {
					return err
				}
			case *enterprisev1.DNSProvider_Spec_Postgresql_:
				typ := spec.GetPostgresql()
				if err := s.validateGenStr(typ.Username, true, "username"); err != nil {
					return err
				}
				if err := s.validateSecretOwner(ctx, typ.Password); err != nil {
					return err
				}

				if !govalidator.IsHost(typ.Host) {
					return grpcutils.InvalidArg("Invalid host: %s", typ.Host)
				}

				if !govalidator.IsPort(fmt.Sprintf("%d", typ.Port)) {
					return grpcutils.InvalidArg("Invalid port: %d", typ.Port)
				}

				if err := s.validateGenStr(typ.Database, true, "database"); err != nil {
					return err
				}

			case *enterprisev1.DNSProvider_Spec_Redis_:
				typ := spec.GetRedis()
				if err := s.validateGenStr(typ.Username, true, "username"); err != nil {
					return err
				}
				if err := s.validateSecretOwner(ctx, typ.Password); err != nil {
					return err
				}

				if !govalidator.IsHost(typ.Host) {
					return grpcutils.InvalidArg("Invalid host: %s", typ.Host)
				}

				if !govalidator.IsPort(fmt.Sprintf("%d", typ.Port)) {
					return grpcutils.InvalidArg("Invalid port: %d", typ.Port)
				}

			case *enterprisev1.DNSProvider_Spec_S3_:
				typ := spec.GetS3()
				if err := s.validateGenStr(typ.AccessKeyID, true, "accessKeyID"); err != nil {
					return err
				}
				if err := s.validateSecretOwner(ctx, typ.SecretAccessKey); err != nil {
					return err
				}

				if !govalidator.IsHost(typ.Endpoint) || !govalidator.IsURL(typ.Endpoint) {
					return grpcutils.InvalidArg("Invalid endpoint: %s", typ.Endpoint)
				}

				if err := s.validateGenStr(typ.Region, false, "region"); err != nil {
					return err
				}
			case *enterprisev1.DNSProvider_Spec_ContainerRegistry_:
				typ := spec.GetContainerRegistry()
				if err := s.validateGenStr(typ.Username, true, "username"); err != nil {
					return err
				}
				if err := s.validateSecretOwner(ctx, typ.Password); err != nil {
					return err
				}

				if !govalidator.IsHost(typ.Server) {
					return grpcutils.InvalidArg("Invalid server: %s", typ.Server)
				}

				if err := s.validateGenStr(typ.Namespace, true, "namespace"); err != nil {
					return err
				}

		*/
	default:
		return grpcutils.InvalidArg("You must set DNSProvider type")
	}

	return nil
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
