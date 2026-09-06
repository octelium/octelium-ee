import * as grpcWeb from "@protobuf-ts/grpcweb-transport";

import { getDomain, isDev } from "..";
import * as AccessC from "../../apis/accessv1/accessv1.client";
import * as AuthGRPC from "../../apis/authv1/authv1.client";
import * as CoreC from "../../apis/corev1/corev1.client";
import * as EnterpriseC from "../../apis/enterprisev1/enterprisev1.client";
import * as UserC from "../../apis/userv1/userv1.client";
import * as VisibilityCoreC from "../../apis/visibilityv1/core/vcorev1.client";
import * as VisibilityAccessC from "../../apis/visibilityv1/access/vaccessv1.client";
import * as VisibilityEnterpriseC from "../../apis/visibilityv1/enterprise/venterprisev1.client";
import * as VisibilityMetricsC from "../../apis/visibilityv1/metrics/vmetricsv1.client";
import * as VisibilityLLMC from "../../apis/visibilityv1/llm/vllmv1.client";
import * as VisibilityC from "../../apis/visibilityv1/visibilityv1.client";

export const getTransport = () => {
  const domain = getDomain();
  const scheme = location.protocol === "https:" ? "https" : "http";

  let baseUrl = `${scheme}://octelium-api.${domain}`;

  if (isDev()) {
    baseUrl = `http://${window.location.host}`;
  }

  return new grpcWeb.GrpcWebFetchTransport({
    baseUrl,

    fetchInit: {
      credentials: "include",
    },
  });
};

export const getClientCore = (): CoreC.MainServiceClient => {
  return new CoreC.MainServiceClient(getTransport());
};

export const getClientEnterprise = (): EnterpriseC.MainServiceClient => {
  return new EnterpriseC.MainServiceClient(getTransport());
};

export const getClientAccess = (): AccessC.MainServiceClient => {
  return new AccessC.MainServiceClient(getTransport());
};

export const getClientUser = (): UserC.MainServiceClient => {
  return new UserC.MainServiceClient(getTransport());
};

export const getClientVisibilityAccessLog =
  (): VisibilityC.AccessLogServiceClient => {
    return new VisibilityC.AccessLogServiceClient(getTransport());
  };

export const getClientVisibilityCore =
  (): VisibilityCoreC.ResourceServiceClient => {
    return new VisibilityCoreC.ResourceServiceClient(getTransport());
  };

export const getClientVisibilityAccess =
  (): VisibilityAccessC.ResourceServiceClient => {
    return new VisibilityAccessC.ResourceServiceClient(getTransport());
  };

export const getClientVisibilityEnterprise =
  (): VisibilityEnterpriseC.ResourceServiceClient => {
    return new VisibilityEnterpriseC.ResourceServiceClient(getTransport());
  };

export const getClientVisibilityAuthenticationLog =
  (): VisibilityC.AuthenticationLogServiceClient => {
    return new VisibilityC.AuthenticationLogServiceClient(getTransport());
  };

export const getClientVisibilityAuditLog =
  (): VisibilityC.AuditLogServiceClient => {
    return new VisibilityC.AuditLogServiceClient(getTransport());
  };

export const getClientVisibilityComponentLog =
  (): VisibilityC.ComponentLogServiceClient => {
    return new VisibilityC.ComponentLogServiceClient(getTransport());
  };

export const getClientAuth = (): AuthGRPC.MainServiceClient => {
  return new AuthGRPC.MainServiceClient(getTransport());
};

export const getClientPolicyPortal =
  (): EnterpriseC.PolicyPortalServiceClient => {
    return new EnterpriseC.PolicyPortalServiceClient(getTransport());
  };

export const getClientCluster = (): EnterpriseC.ClusterServiceClient => {
  return new EnterpriseC.ClusterServiceClient(getTransport());
};

let visibilityMetricsClient: VisibilityMetricsC.MetricsServiceClient | undefined;

export const getClientVisibilityMetrics =
  (): VisibilityMetricsC.MetricsServiceClient => {
    visibilityMetricsClient ??= new VisibilityMetricsC.MetricsServiceClient(
      getTransport(),
    );
    return visibilityMetricsClient;
  };

let visibilityLLMClient: VisibilityLLMC.LLMServiceClient | undefined;

export const getClientVisibilityLLM = (): VisibilityLLMC.LLMServiceClient => {
  visibilityLLMClient ??= new VisibilityLLMC.LLMServiceClient(getTransport());
  return visibilityLLMClient;
};

export const refetchIntervalChart = 15000;
