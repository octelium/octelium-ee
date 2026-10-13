import * as grpcWeb from "@protobuf-ts/grpcweb-transport";

import { getDomain, isDev } from "..";
import * as AccessC from "../../apis/accessv1/accessv1.client";
import * as AuthGRPC from "../../apis/authv1/authv1.client";
import * as CordiumC from "../../apis/cordiumv1/cordiumv1.client";
import * as CoreC from "../../apis/corev1/corev1.client";
import * as EnterpriseC from "../../apis/enterprisev1/enterprisev1.client";
import * as UserC from "../../apis/userv1/userv1.client";
import * as VisibilityCoreC from "../../apis/visibilityv1/core/vcorev1.client";
import * as VisibilityAccessC from "../../apis/visibilityv1/access/vaccessv1.client";
import * as VisibilityCordiumC from "../../apis/visibilityv1/cordium/vcordiumv1.client";
import * as VisibilityEnterpriseC from "../../apis/visibilityv1/enterprise/venterprisev1.client";
import * as VisibilityMetricsC from "../../apis/visibilityv1/metrics/vmetricsv1.client";
import * as VisibilityLLMC from "../../apis/visibilityv1/llm/vllmv1.client";
import * as VisibilityC from "../../apis/visibilityv1/visibilityv1.client";

let transport: grpcWeb.GrpcWebFetchTransport | undefined;

export const getTransport = () => {
  if (transport) return transport;
  const domain = getDomain();
  const scheme = location.protocol === "https:" ? "https" : "http";

  let baseUrl = `${scheme}://octelium-api.${domain}`;

  if (isDev()) {
    baseUrl = `http://${window.location.host}`;
  }

  transport = new grpcWeb.GrpcWebFetchTransport({
    baseUrl,

    fetchInit: {
      credentials: "include",
    },
  });
  return transport;
};

let coreClient: CoreC.MainServiceClient | undefined;

export const getClientCore = (): CoreC.MainServiceClient => {
  coreClient ??= new CoreC.MainServiceClient(getTransport());
  return coreClient;
};

let enterpriseClient: EnterpriseC.MainServiceClient | undefined;

export const getClientEnterprise = (): EnterpriseC.MainServiceClient => {
  enterpriseClient ??= new EnterpriseC.MainServiceClient(getTransport());
  return enterpriseClient;
};

let accessClient: AccessC.MainServiceClient | undefined;

export const getClientAccess = (): AccessC.MainServiceClient => {
  accessClient ??= new AccessC.MainServiceClient(getTransport());
  return accessClient;
};

let userClient: UserC.MainServiceClient | undefined;

export const getClientUser = (): UserC.MainServiceClient => {
  userClient ??= new UserC.MainServiceClient(getTransport());
  return userClient;
};

let visibilityAccessLogClient: VisibilityC.AccessLogServiceClient | undefined;

export const getClientVisibilityAccessLog =
  (): VisibilityC.AccessLogServiceClient => {
    visibilityAccessLogClient ??= new VisibilityC.AccessLogServiceClient(
      getTransport(),
    );
    return visibilityAccessLogClient;
  };

let visibilityCoreClient: VisibilityCoreC.ResourceServiceClient | undefined;

export const getClientVisibilityCore =
  (): VisibilityCoreC.ResourceServiceClient => {
    visibilityCoreClient ??= new VisibilityCoreC.ResourceServiceClient(
      getTransport(),
    );
    return visibilityCoreClient;
  };

let visibilityAccessClient: VisibilityAccessC.ResourceServiceClient | undefined;

export const getClientVisibilityAccess =
  (): VisibilityAccessC.ResourceServiceClient => {
    visibilityAccessClient ??= new VisibilityAccessC.ResourceServiceClient(
      getTransport(),
    );
    return visibilityAccessClient;
  };

let visibilityEnterpriseClient:
  VisibilityEnterpriseC.ResourceServiceClient | undefined;

export const getClientVisibilityEnterprise =
  (): VisibilityEnterpriseC.ResourceServiceClient => {
    visibilityEnterpriseClient ??=
      new VisibilityEnterpriseC.ResourceServiceClient(getTransport());
    return visibilityEnterpriseClient;
  };

let visibilityCordiumClient:
  VisibilityCordiumC.ResourceServiceClient | undefined;

export const getClientVisibilityCordium =
  (): VisibilityCordiumC.ResourceServiceClient => {
    visibilityCordiumClient ??= new VisibilityCordiumC.ResourceServiceClient(
      getTransport(),
    );
    return visibilityCordiumClient;
  };

let visibilityAuthenticationLogClient:
  VisibilityC.AuthenticationLogServiceClient | undefined;

export const getClientVisibilityAuthenticationLog =
  (): VisibilityC.AuthenticationLogServiceClient => {
    visibilityAuthenticationLogClient ??=
      new VisibilityC.AuthenticationLogServiceClient(getTransport());
    return visibilityAuthenticationLogClient;
  };

let visibilityAuditLogClient: VisibilityC.AuditLogServiceClient | undefined;

export const getClientVisibilityAuditLog =
  (): VisibilityC.AuditLogServiceClient => {
    visibilityAuditLogClient ??= new VisibilityC.AuditLogServiceClient(
      getTransport(),
    );
    return visibilityAuditLogClient;
  };

let visibilityComponentLogClient:
  VisibilityC.ComponentLogServiceClient | undefined;

export const getClientVisibilityComponentLog =
  (): VisibilityC.ComponentLogServiceClient => {
    visibilityComponentLogClient ??= new VisibilityC.ComponentLogServiceClient(
      getTransport(),
    );
    return visibilityComponentLogClient;
  };

let visibilityClusterClient: VisibilityC.ClusterServiceClient | undefined;

export const getClientVisibilityCluster =
  (): VisibilityC.ClusterServiceClient => {
    visibilityClusterClient ??= new VisibilityC.ClusterServiceClient(
      getTransport(),
    );
    return visibilityClusterClient;
  };

let authClient: AuthGRPC.MainServiceClient | undefined;

export const getClientAuth = (): AuthGRPC.MainServiceClient => {
  authClient ??= new AuthGRPC.MainServiceClient(getTransport());
  return authClient;
};

let policyPortalClient: EnterpriseC.PolicyPortalServiceClient | undefined;

export const getClientPolicyPortal =
  (): EnterpriseC.PolicyPortalServiceClient => {
    policyPortalClient ??= new EnterpriseC.PolicyPortalServiceClient(
      getTransport(),
    );
    return policyPortalClient;
  };

let clusterClient: EnterpriseC.ClusterServiceClient | undefined;

export const getClientCluster = (): EnterpriseC.ClusterServiceClient => {
  clusterClient ??= new EnterpriseC.ClusterServiceClient(getTransport());
  return clusterClient;
};

let agentClient: EnterpriseC.AgentServiceClient | undefined;

export const getClientAgent = (): EnterpriseC.AgentServiceClient => {
  agentClient ??= new EnterpriseC.AgentServiceClient(getTransport());
  return agentClient;
};

let cordiumClient: CordiumC.MainServiceClient | undefined;

export const getClientCordium = (): CordiumC.MainServiceClient => {
  cordiumClient ??= new CordiumC.MainServiceClient(getTransport());
  return cordiumClient;
};

let visibilityMetricsClient:
  VisibilityMetricsC.MetricsServiceClient | undefined;

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

export const chartRefreshMillis = 15000;

export const refetchIntervalChart = () =>
  document.hidden ? (false as const) : chartRefreshMillis;
