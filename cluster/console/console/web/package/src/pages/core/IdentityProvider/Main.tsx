import * as CoreC from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceInfoIcon, ResourceMainInfo } from "@/pages/utils/types";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  MutedValue,
  OverviewNotice,
  TonePill,
} from "@/components/ResourceLayout/overview";
import {
  Ban,
  Building2,
  KeyRound,
  ListChecks,
  Lock,
  LogIn,
  Mail,
  Workflow,
} from "lucide-react";
import { FaGithub } from "react-icons/fa6";
import { Switch } from "@mantine/core";
import { twMerge } from "@/lib/utils";
import { getType } from "./List";

export const ItemInfo = (props: { item: CoreC.IdentityProvider }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <>
      <InfoItem title="Type">
        <Label>{getType(item)}</Label>
      </InfoItem>
      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={!item.spec!.isDisabled}
            onChange={(v) => {
              const next = CoreC.IdentityProvider.clone(item);
              next.spec!.isDisabled = !v.currentTarget.checked;
              mutationUpdate.mutate(next);
            }}
          />
        </div>
      </InfoItem>
    </>
  );
};

export default (props: { item: CoreC.IdentityProvider }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

const secretLabel = (name?: string) =>
  name ? (
    <ResourceListLabel
      itemRef={ObjectReference.create({
        apiVersion: "core/v1",
        kind: "Secret",
        name,
      })}
    />
  ) : (
    <MutedValue>Not set</MutedValue>
  );

const conditionText = (condition?: CoreC.Condition) => {
  const type = condition?.type;
  switch (type?.oneofKind) {
    case "match":
      return type.match;
    case "matchAny":
      return type.matchAny ? "Always" : "Never";
    case "not":
      return `not (${type.not})`;
    case "all":
      return `All of ${type.all.of.length} conditions`;
    case "any":
      return `Any of ${type.any.of.length} conditions`;
    case "none":
      return `None of ${type.none.of.length} conditions`;
    case "opa":
      return "OPA (Rego) policy";
    default:
      return "Always";
  }
};

const typeIcon = (item: CoreC.IdentityProvider): ResourceInfoIcon => {
  switch (item.spec?.type.oneofKind) {
    case "github":
      return FaGithub as ResourceInfoIcon;
    case "saml":
      return Building2;
    case "oidcIdentityToken":
      return Workflow;
    default:
      return KeyRound;
  }
};

const providerItems = (item: CoreC.IdentityProvider, group: string) => {
  const type = item.spec!.type;
  const rows: { label: string; value: React.ReactNode; span?: "full" }[] = [];

  if (type.oneofKind === "github") {
    rows.push(
      { label: "Client ID", value: <CopyText value={type.github.clientID} /> },
      {
        label: "Client secret",
        value: secretLabel(
          type.github.clientSecret?.type.oneofKind === "fromSecret"
            ? type.github.clientSecret.type.fromSecret
            : undefined,
        ),
      },
    );
  }

  if (type.oneofKind === "oidc") {
    rows.push(
      { label: "Issuer URL", value: <CopyText value={type.oidc.issuerURL} />, span: "full" },
      { label: "Client ID", value: <CopyText value={type.oidc.clientID} /> },
      {
        label: "Client secret",
        value: secretLabel(
          type.oidc.clientSecret?.type.oneofKind === "fromSecret"
            ? type.oidc.clientSecret.type.fromSecret
            : undefined,
        ),
      },
      {
        label: "Identifier claim",
        value: type.oidc.identifierClaim || <MutedValue>Default</MutedValue>,
      },
      {
        label: "Scopes",
        value: type.oidc.scopes.length ? (
          <div className="flex flex-wrap gap-1">
            {type.oidc.scopes.map((scope) => (
              <Label key={scope} size="sm" outlined>
                {scope}
              </Label>
            ))}
          </div>
        ) : (
          <MutedValue>Default</MutedValue>
        ),
      },
      {
        label: "Checks",
        value: (
          <div className="flex flex-wrap gap-1">
            <TonePill tone={type.oidc.checkEmailVerified ? "success" : "neutral"}>
              Verified email {type.oidc.checkEmailVerified ? "required" : "not required"}
            </TonePill>
            {type.oidc.useUserInfoEndpoint && (
              <TonePill tone="info">UserInfo endpoint</TonePill>
            )}
          </div>
        ),
        span: "full",
      },
    );
  }

  if (type.oneofKind === "saml") {
    rows.push(
      {
        label: "Metadata",
        value:
          type.saml.metadataType.oneofKind === "metadataURL" ? (
            <CopyText value={type.saml.metadataType.metadataURL} />
          ) : type.saml.metadataType.oneofKind === "metadata" ? (
            "Inline metadata"
          ) : (
            <MutedValue>Not set</MutedValue>
          ),
        span: "full",
      },
      ...(type.saml.entityID
        ? [{ label: "Entity ID", value: <CopyText value={type.saml.entityID} />, span: "full" as const }]
        : []),
      {
        label: "Identifier attribute",
        value: type.saml.identifierAttribute || <MutedValue>Default</MutedValue>,
      },
      {
        label: "Force re-authentication",
        value: type.saml.forceAuthn ? "Enabled" : <MutedValue>Disabled</MutedValue>,
      },
    );
  }

  if (type.oneofKind === "oidcIdentityToken") {
    const source = type.oidcIdentityToken.type;
    rows.push(
      {
        label: "Keys",
        value:
          source.oneofKind === "issuerURL" ? (
            <CopyText value={source.issuerURL} />
          ) : source.oneofKind === "jwksURL" ? (
            <CopyText value={source.jwksURL} />
          ) : source.oneofKind === "jwksContent" ? (
            "Inline JWKS"
          ) : (
            <MutedValue>Not set</MutedValue>
          ),
        span: "full",
      },
      {
        label: "Issuer",
        value: type.oidcIdentityToken.issuer || <MutedValue>Any</MutedValue>,
      },
      {
        label: "Audience",
        value: type.oidcIdentityToken.audience || <MutedValue>Any</MutedValue>,
      },
    );
  }

  return rows.map((row) => ({ ...row, group }));
};

const aalLabel = (aal: CoreC.IdentityProvider_Spec_AALRule_AAL) =>
  CoreC.IdentityProvider_Spec_AALRule_AAL[aal] ?? "Unknown";

export const MainInfo = (props: {
  item: CoreC.IdentityProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const spec = item.spec!;
  const Icon = typeIcon(item);

  return {
    header:
      spec.isDisabled || item.status?.isLocked ? (
        <div className="flex flex-col gap-2 border-b border-slate-200 bg-white px-4 py-3">
          {spec.isDisabled && (
            <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
              Users cannot sign in with this IdentityProvider.
            </OverviewNotice>
          )}
          {item.status?.isLocked && (
            <OverviewNotice tone="danger" icon={Lock} title="Locked by the Cluster.">
              Authentications through this IdentityProvider are rejected.
            </OverviewNotice>
          )}
        </div>
      ) : undefined,
    groupOrder: [
      { title: "Provider", icon: Icon },
      { title: "Authentication rules", icon: ListChecks },
    ],
    items: [
      {
        label: "Type",
        primary: true,
        icon: Icon,
        value: <Label>{getType(item)}</Label>,
      },
      {
        label: "Login button",
        primary: true,
        icon: LogIn,
        value: spec.displayName || <MutedValue>Default</MutedValue>,
      },
      {
        label: "Email matching",
        primary: true,
        icon: Mail,
        hint: "Link Users by their email address",
        value: spec.disableEmailAsIdentity ? (
          <MutedValue>Disabled</MutedValue>
        ) : (
          <span className="font-semibold">Enabled</span>
        ),
      },
      {
        label: "Rules",
        primary: true,
        icon: ListChecks,
        value: (
          <span className="font-semibold tabular-nums">
            {spec.aalRules.length + spec.postAuthenticationRules.length}
          </span>
        ),
      },
      ...providerItems(item, "Provider"),
      {
        label: "Assurance level",
        group: "Authentication rules",
        span: "full" as const,
        value: spec.aalRules.length ? (
          <ul className="flex flex-col gap-1">
            {spec.aalRules.map((rule, index) => (
              <li key={index} className="flex flex-wrap items-center gap-1.5">
                <TonePill tone="success">{aalLabel(rule.aal)}</TonePill>
                <code className="text-xs text-slate-600 [overflow-wrap:anywhere]">
                  {conditionText(rule.condition)}
                </code>
              </li>
            ))}
          </ul>
        ) : (
          <MutedValue>No rules; the default assurance level applies.</MutedValue>
        ),
      },
      {
        label: "Post-authentication",
        group: "Authentication rules",
        span: "full" as const,
        value: spec.postAuthenticationRules.length ? (
          <ul className="flex flex-col gap-1">
            {spec.postAuthenticationRules.map((rule, index) => (
              <li key={index} className="flex flex-wrap items-center gap-1.5">
                <TonePill
                  tone={
                    rule.effect ===
                    CoreC.IdentityProvider_Spec_PostAuthenticationRule_Effect.DENY
                      ? "danger"
                      : "success"
                  }
                >
                  {rule.effect ===
                  CoreC.IdentityProvider_Spec_PostAuthenticationRule_Effect.DENY
                    ? "Deny"
                    : "Allow"}
                </TonePill>
                <code className="text-xs text-slate-600 [overflow-wrap:anywhere]">
                  {conditionText(rule.condition)}
                </code>
              </li>
            ))}
          </ul>
        ) : (
          <MutedValue>No rules; every successful login is accepted.</MutedValue>
        ),
      },
    ],
  };
};
