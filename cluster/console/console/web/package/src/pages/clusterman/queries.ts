import {
  ClusterConfig_Status_UpgradeRequest_State,
  GetClusterInfoRequest,
  GetClusterInfoResponse,
  GetLicenseRequest,
  UpgradeClusterRequest,
} from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { getClientCluster, getClientEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Boxes, Cpu, LucideIcon, Sparkles } from "lucide-react";

export const clustermanKeys = {
  info: ["clusterInfo"],
  config: ["clusterman", "main", "getCluster"],
  license: ["clusterman", "license"],
};

export type PackageKey = "core" | "packageEnterprise" | "packageCordium";

export type PackageInfo = {
  currentVersion: string;
  latestVersion: string;
  canUpgrade: boolean;
  setAt?: Timestamp;
};

export type PackageMeta = {
  key: PackageKey;
  label: string;
  description: string;
  icon: LucideIcon;
};

export const PACKAGES: PackageMeta[] = [
  {
    key: "core",
    label: "Core",
    description: "The core Cluster runtime and control plane",
    icon: Cpu,
  },
  {
    key: "packageEnterprise",
    label: "Enterprise",
    description: "Enterprise features and external integrations",
    icon: Boxes,
  },
  {
    key: "packageCordium",
    label: "Cordium",
    description: "The Cordium sandbox platform package",
    icon: Sparkles,
  },
];

export const packageInfo = (
  data: GetClusterInfoResponse | undefined,
  key: PackageKey,
): PackageInfo | undefined => data?.[key];

export const upgradableCount = (data?: GetClusterInfoResponse) =>
  PACKAGES.filter((item) => packageInfo(data, item.key)?.canUpgrade).length;

export const isUpgradeActive = (
  state?: ClusterConfig_Status_UpgradeRequest_State,
) =>
  state === ClusterConfig_Status_UpgradeRequest_State.UPGRADE_REQUESTED ||
  state === ClusterConfig_Status_UpgradeRequest_State.UPGRADING;

export const useClusterVersionInfo = () =>
  useQuery({
    queryKey: clustermanKeys.info,
    queryFn: async () => {
      const { response } = await getClientCluster().getClusterInfo(
        GetClusterInfoRequest.create({}),
      );
      return response;
    },
    refetchInterval: 60_000,
  });

export const useClusterConfig = () =>
  useQuery({
    queryKey: clustermanKeys.config,
    queryFn: async () => {
      const { response } = await getClientEnterprise().getClusterConfig({});
      return response;
    },
    refetchInterval: (query) =>
      isUpgradeActive(query.state.data?.status?.upgradeRequest?.state)
        ? 5_000
        : 30_000,
  });

export const useLicense = () =>
  useQuery({
    queryKey: clustermanKeys.license,
    queryFn: async () => {
      const { response } = await getClientCluster().getLicense(
        GetLicenseRequest.create({}),
      );
      return response;
    },
    retry: 1,
    refetchInterval: 300_000,
  });

export const selectedPackages = (req: UpgradeClusterRequest): PackageKey[] =>
  PACKAGES.filter((item) => !!req.request?.[item.key]).map((item) => item.key);
