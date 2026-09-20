import {
  Authenticator_Spec_State,
  Authenticator_Status_Type,
  Credential_Spec_Type,
  Device_Spec_State,
  Device_Status_OSType,
  IdentityProvider_Status_Type,
  Service_Spec_Mode,
  Session_Spec_State,
  Session_Status_Type,
  User_Spec_Type,
} from "@/apis/corev1/corev1";
import {
  Certificate_Spec_Mode,
  Certificate_Status_Issuance_State,
  DirectoryProvider_Status_Synchronization_State,
  SecretStore_Status_Type,
} from "@/apis/enterprisev1/enterprisev1";
import {
  Integration_Status_State,
  Integration_Status_Type,
  Request_Spec_Urgency,
  Request_Status_State_Status,
} from "@/apis/accessv1/accessv1";
import { getAPI, Resource } from "@/utils/pb";
import {
  BookKey,
  Boxes,
  ClipboardCheck,
  Crown,
  DoorClosed,
  Fingerprint,
  Folder,
  Globe,
  Globe2,
  Inbox,
  KeyRound,
  LaptopMinimal,
  Layers,
  LockKeyhole,
  LockOpen,
  LucideIcon,
  PanelTop,
  Plug,
  Shield,
  ShieldCheck,
  Telescope,
  Terminal,
  User,
  Users,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  "core/User": User,
  "core/Group": Users,
  "core/Session": Terminal,
  "core/Device": LaptopMinimal,
  "core/Service": PanelTop,
  "core/Namespace": Boxes,
  "core/Policy": Shield,
  "core/Credential": LockOpen,
  "core/IdentityProvider": Fingerprint,
  "core/Authenticator": LockKeyhole,
  "core/Secret": KeyRound,
  "core/Gateway": DoorClosed,
  "core/Region": Globe,
  "access/Policy": Shield,
  "access/Catalog": Layers,
  "access/Request": Inbox,
  "access/Review": ClipboardCheck,
  "access/Secret": KeyRound,
  "access/Integration": Plug,
  "enterprise/Certificate": ShieldCheck,
  "enterprise/CertificateIssuer": Crown,
  "enterprise/DirectoryProvider": Folder,
  "enterprise/SecretStore": BookKey,
  "enterprise/Secret": KeyRound,
  "enterprise/CollectorExporter": Telescope,
  "enterprise/DNSProvider": Globe2,
};

export const resourceIcon = (api: string, kind: string): LucideIcon =>
  ICONS[`${api}/${kind}`] ?? Boxes;

export type ResourceFacts = {
  primary?: string;
  chips: string[];
  isDisabled?: boolean;
};

const label = (
  values: Record<number, string>,
  value: number | undefined,
): string | undefined => {
  if (!value) return undefined;
  return values[value]?.replace(/_/g, " ");
};

const clean = (facts: ResourceFacts): ResourceFacts => ({
  ...facts,
  chips: facts.chips.filter((chip) => !!chip),
});

const EXTRACTORS: Record<string, (item: any) => ResourceFacts> = {
  "core/User": (item) => ({
    primary: item.spec?.email || undefined,
    chips: [
      label(User_Spec_Type, item.spec?.type),
      item.spec?.groups?.length
        ? `${item.spec.groups.length} group${item.spec.groups.length === 1 ? "" : "s"}`
        : undefined,
    ].filter(Boolean) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "core/Service": (item) => ({
    primary: item.status?.primaryHostname || undefined,
    chips: [
      label(Service_Spec_Mode, item.spec?.mode),
      item.spec?.isPublic ? "PUBLIC" : undefined,
      item.spec?.isAnonymous ? "ANONYMOUS" : undefined,
      item.status?.namespaceRef?.name,
    ].filter(Boolean) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "core/Device": (item) => ({
    primary: item.status?.hostname || item.status?.userRef?.name || undefined,
    chips: [
      label(Device_Status_OSType, item.status?.osType),
      label(Device_Spec_State, item.spec?.state),
    ].filter(Boolean) as string[],
  }),
  "core/Session": (item) => ({
    primary: item.status?.userRef?.name || undefined,
    chips: [
      label(Session_Status_Type, item.status?.type),
      label(Session_Spec_State, item.spec?.state),
      item.status?.isConnected ? "CONNECTED" : undefined,
    ].filter(Boolean) as string[],
  }),
  "core/Credential": (item) => ({
    primary: item.spec?.user || undefined,
    chips: [label(Credential_Spec_Type, item.spec?.type)].filter(
      Boolean,
    ) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "core/IdentityProvider": (item) => ({
    chips: [label(IdentityProvider_Status_Type, item.status?.type)].filter(
      Boolean,
    ) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "core/Authenticator": (item) => ({
    primary: item.status?.userRef?.name || undefined,
    chips: [
      label(Authenticator_Status_Type, item.status?.type),
      label(Authenticator_Spec_State, item.spec?.state),
    ].filter(Boolean) as string[],
  }),
  "core/Policy": (item) => ({
    chips: item.spec?.rules?.length
      ? [
          `${item.spec.rules.length} rule${item.spec.rules.length === 1 ? "" : "s"}`,
        ]
      : [],
    isDisabled: item.spec?.isDisabled,
  }),
  "core/Gateway": (item) => ({
    primary: item.status?.regionRef?.name || undefined,
    chips: [],
  }),
  "access/Policy": (item) => ({
    chips: item.spec?.rules?.length
      ? [
          `${item.spec.rules.length} rule${item.spec.rules.length === 1 ? "" : "s"}`,
        ]
      : [],
    isDisabled: item.spec?.isDisabled,
  }),
  "access/Request": (item) => ({
    primary: item.status?.userRef?.name || undefined,
    chips: [
      label(Request_Status_State_Status, item.status?.state?.status),
      label(Request_Spec_Urgency, item.spec?.urgency),
    ].filter(Boolean) as string[],
  }),
  "access/Integration": (item) => ({
    chips: [
      label(Integration_Status_Type, item.status?.type),
      label(Integration_Status_State, item.status?.state),
    ].filter(Boolean) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "enterprise/Certificate": (item) => ({
    primary: item.status?.info?.commonName || undefined,
    chips: [
      label(Certificate_Spec_Mode, item.spec?.mode),
      label(Certificate_Status_Issuance_State, item.status?.issuance?.state),
    ].filter(Boolean) as string[],
  }),
  "enterprise/SecretStore": (item) => ({
    chips: [label(SecretStore_Status_Type, item.status?.type)].filter(
      Boolean,
    ) as string[],
  }),
  "enterprise/DirectoryProvider": (item) => ({
    chips: [
      label(
        DirectoryProvider_Status_Synchronization_State,
        item.status?.synchronization?.state,
      ),
    ].filter(Boolean) as string[],
    isDisabled: item.spec?.isDisabled,
  }),
  "enterprise/CollectorExporter": (item) => ({
    chips: [],
    isDisabled: item.spec?.isDisabled,
  }),
};

export const resourceFacts = (item: Resource): ResourceFacts => {
  const key = `${getAPI(item) ?? ""}/${item.kind}`;
  const extractor = EXTRACTORS[key];
  if (!extractor) return { chips: [] };
  return clean(extractor(item));
};
