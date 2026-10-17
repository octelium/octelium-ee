import * as AccessP from "@/apis/accessv1/accessv1";
import {
  ListIntegrationBindingOptions,
  ListIntegrationIdentityOptions,
  ListPolicyOptions,
  ListSecretOptions,
} from "@/apis/visibilityv1/access/vaccessv1";
import {
  CommonListOptions,
  CommonListOptions_OrderBy_Mode,
  CommonListOptions_OrderBy_Type,
} from "@/apis/visibilityv1/meta/vmetav1";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";

const recentFirst = {
  type: CommonListOptions_OrderBy_Type.CREATED_AT,
  mode: CommonListOptions_OrderBy_Mode.DESC,
};

const identityOf = (item: AccessP.Integration) =>
  item.metadata?.uid || item.metadata?.name || "";

export const useIntegrationDeliveries = (
  item: AccessP.Integration,
  opts?: { failingOnly?: boolean; itemsPerPage?: number },
) =>
  useQuery({
    queryKey: [
      "access",
      "integration",
      identityOf(item),
      "deliveries",
      opts?.failingOnly ?? false,
      opts?.itemsPerPage ?? 20,
    ],
    enabled: !!identityOf(item),
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listIntegrationBinding(
          ListIntegrationBindingOptions.create({
            common: CommonListOptions.create({
              itemsPerPage: opts?.itemsPerPage ?? 20,
              orderBy: recentFirst,
            }),
            integrationRef: getResourceRef(item),
            isFailing: opts?.failingOnly ?? false,
          }),
          { abort: signal },
        )
      ).response,
  });

export const useIntegrationDeliverySummary = (item: AccessP.Integration) =>
  useQuery({
    queryKey: ["access", "integration", identityOf(item), "deliverySummary"],
    enabled: !!identityOf(item),
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().getIntegrationBindingSummary(
          { integrationRef: getResourceRef(item) },
          { abort: signal },
        )
      ).response,
  });

export const useIntegrationIdentities = (item: AccessP.Integration) =>
  useQuery({
    queryKey: ["access", "integration", identityOf(item), "identities"],
    enabled: !!identityOf(item),
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listIntegrationIdentity(
          ListIntegrationIdentityOptions.create({
            common: CommonListOptions.create({
              itemsPerPage: 50,
              orderBy: recentFirst,
            }),
            integrationRef: getResourceRef(item),
          }),
          { abort: signal },
        )
      ).response,
  });

export type IntegrationUsage = {
  policy: AccessP.Policy;
  rule: AccessP.Policy_Spec_Rule;
  stepName?: string;
  surface: AccessP.Policy_Spec_Rule_Surface;
  isNotification: boolean;
};

const surfaceTargets = (
  surface: AccessP.Policy_Spec_Rule_Surface,
  item: AccessP.Integration,
) => {
  const ref = surface.destination?.integrationRef;
  if (!ref) return false;
  if (ref.uid && item.metadata?.uid) return ref.uid === item.metadata.uid;
  return !!ref.name && ref.name === item.metadata?.name;
};

export const getIntegrationUsages = (
  policies: AccessP.Policy[],
  item: AccessP.Integration,
): IntegrationUsage[] => {
  const ret: IntegrationUsage[] = [];

  for (const policy of policies) {
    for (const rule of policy.spec?.rules ?? []) {
      if (rule.action?.type.oneofKind === "review") {
        rule.action.type.review.steps.forEach((step, index) => {
          for (const surface of step.surfaces) {
            if (surfaceTargets(surface, item)) {
              ret.push({
                policy,
                rule,
                stepName: step.name || `step-${index + 1}`,
                surface,
                isNotification: false,
              });
            }
          }
        });
      }

      for (const surface of rule.notifications) {
        if (surfaceTargets(surface, item)) {
          ret.push({ policy, rule, surface, isNotification: true });
        }
      }
    }
  }

  return ret;
};

export const useAccessPolicies = () =>
  useQuery({
    queryKey: ["access", "policies", "all"],
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listPolicy(
          ListPolicyOptions.create({
            common: CommonListOptions.create({ itemsPerPage: 500 }),
          }),
          { abort: signal },
        )
      ).response,
  });

export const useAccessSecretNames = () =>
  useQuery({
    queryKey: ["access", "secrets", "names"],
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      new Set(
        (
          await getClientVisibilityAccess().listSecret(
            ListSecretOptions.create({
              common: CommonListOptions.create({ itemsPerPage: 1000 }),
            }),
            { abort: signal },
          )
        ).response.items.map((secret) => secret.metadata?.name ?? ""),
      ),
  });
