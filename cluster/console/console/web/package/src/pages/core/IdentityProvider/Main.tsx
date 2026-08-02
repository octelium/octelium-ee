import * as CoreC from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { Switch } from "@mantine/core";
import { twMerge } from "tailwind-merge";
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

const ProviderConfiguration = (props: { item: CoreC.IdentityProvider }) => {
  const type = props.item.spec!.type;

  if (type.oneofKind === "github") {
    return (
      <ResourceListLabel label="Client ID">
        <CopyText value={type.github.clientID} />
      </ResourceListLabel>
    );
  }

  if (type.oneofKind === "oidc") {
    return (
      <div className="flex flex-wrap gap-1">
        {type.oidc.clientID && (
          <ResourceListLabel label="Client ID">
            <CopyText value={type.oidc.clientID} />
          </ResourceListLabel>
        )}
        {type.oidc.issuerURL && (
          <ResourceListLabel label="Issuer">
            <CopyText value={type.oidc.issuerURL} />
          </ResourceListLabel>
        )}
        {type.oidc.identifierClaim && (
          <ResourceListLabel label="Identifier claim">
            {type.oidc.identifierClaim}
          </ResourceListLabel>
        )}
        {type.oidc.scopes.map((scope) => (
          <ResourceListLabel key={scope} label="Scope">
            {scope}
          </ResourceListLabel>
        ))}
        {type.oidc.checkEmailVerified && (
          <ResourceListLabel>Verified email required</ResourceListLabel>
        )}
        {type.oidc.useUserInfoEndpoint && (
          <ResourceListLabel>UserInfo endpoint enabled</ResourceListLabel>
        )}
      </div>
    );
  }

  if (type.oneofKind === "saml") {
    return (
      <div className="flex flex-wrap gap-1">
        {type.saml.metadataType.oneofKind === "metadataURL" ? (
          <ResourceListLabel label="Metadata URL">
            <CopyText value={type.saml.metadataType.metadataURL} />
          </ResourceListLabel>
        ) : type.saml.metadataType.oneofKind === "metadata" ? (
          <ResourceListLabel>Inline metadata configured</ResourceListLabel>
        ) : null}
        {type.saml.entityID && (
          <ResourceListLabel label="Entity ID">
            <CopyText value={type.saml.entityID} />
          </ResourceListLabel>
        )}
        {type.saml.identifierAttribute && (
          <ResourceListLabel label="Identifier attribute">
            {type.saml.identifierAttribute}
          </ResourceListLabel>
        )}
        {type.saml.forceAuthn && (
          <ResourceListLabel>Forced re-authentication</ResourceListLabel>
        )}
      </div>
    );
  }

  if (type.oneofKind === "oidcIdentityToken") {
    const source = type.oidcIdentityToken.type;
    return (
      <div className="flex flex-wrap gap-1">
        {source.oneofKind === "issuerURL" && (
          <ResourceListLabel label="Issuer URL">
            <CopyText value={source.issuerURL} />
          </ResourceListLabel>
        )}
        {source.oneofKind === "jwksURL" && (
          <ResourceListLabel label="JWKS URL">
            <CopyText value={source.jwksURL} />
          </ResourceListLabel>
        )}
        {source.oneofKind === "jwksContent" && (
          <ResourceListLabel>Inline JWKS configured</ResourceListLabel>
        )}
        {type.oidcIdentityToken.issuer && (
          <ResourceListLabel label="Issuer">
            {type.oidcIdentityToken.issuer}
          </ResourceListLabel>
        )}
        {type.oidcIdentityToken.audience && (
          <ResourceListLabel label="Audience">
            {type.oidcIdentityToken.audience}
          </ResourceListLabel>
        )}
      </div>
    );
  }

  return null;
};

export const MainInfo = (props: {
  item: CoreC.IdentityProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "Type",
        value: <Label>{getType(item)}</Label>,
      },
      ...(item.spec?.displayName
        ? [
            {
              label: "Display name",
              value: item.spec.displayName,
            },
          ]
        : []),
      {
        label: "Active",
        value: (
          <EditItemWrap
            mutation={mutationUpdate}
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  const next = CoreC.IdentityProvider.clone(item);
                  next.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(next);
                }}
              />
            }
          />
        ),
      },
      ...(item.status?.isLocked
        ? [
            {
              label: "Security state",
              value: <span className="font-semibold text-red-600">Locked</span>,
            },
          ]
        : []),
      {
        label: "Provider configuration",
        value: <ProviderConfiguration item={item} />,
        span: "full",
      },
      ...(item.spec!.aalRules.length ||
      item.spec!.postAuthenticationRules.length
        ? [
            {
              label: "Authentication rules",
              value: (
                <div className="flex flex-wrap gap-1">
                  <ResourceListLabel label="Assurance level">
                    {item.spec!.aalRules.length}
                  </ResourceListLabel>
                  <ResourceListLabel label="Post-authentication">
                    {item.spec!.postAuthenticationRules.length}
                  </ResourceListLabel>
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
