import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import {
  getGitProviderConfig,
  getGitProviderTypeLabel,
  spaceRefOf,
  userRefOf,
} from "../utils";

export const MainInfo = (props: {
  item: CordiumP.GitProvider;
}): ResourceMainInfo => {
  const { item } = props;
  const config = getGitProviderConfig(item);
  const typeLabel = getGitProviderTypeLabel(item);
  const userRef = userRefOf(item.status?.userRef);
  const spaceRef = spaceRefOf(item.status?.spaceRef);

  return {
    status: { label: typeLabel, tone: "info" },
    groupOrder: ["GitProvider details", "OAuth2 client"],
    items: [
      {
        label: "Type",
        primary: true,
        value: <Label>{typeLabel}</Label>,
      },
      ...(spaceRef
        ? [{ label: "Space", value: <ResourceListLabel itemRef={spaceRef} /> }]
        : []),
      ...(userRef
        ? [{ label: "User", value: <ResourceListLabel itemRef={userRef} /> }]
        : []),
      ...(config?.clientID
        ? [
            {
              label: "Client ID",
              group: "OAuth2 client",
              value: <CopyText value={config.clientID} />,
            },
          ]
        : []),
      ...(config?.clientSecret
        ? [
            {
              label: "Client secret",
              group: "OAuth2 client",
              value: (
                <ResourceListLabel label="From Secret">
                  {config.clientSecret}
                </ResourceListLabel>
              ),
            },
          ]
        : []),
      ...(config?.authURL
        ? [
            {
              label: "Authorization URL",
              group: "OAuth2 client",
              span: "full" as const,
              value: <CopyText value={config.authURL} />,
            },
          ]
        : []),
      ...(config?.tokenURL
        ? [
            {
              label: "Token URL",
              group: "OAuth2 client",
              span: "full" as const,
              value: <CopyText value={config.tokenURL} />,
            },
          ]
        : []),
      ...((config?.scopes.length ?? 0) > 0
        ? [
            {
              label: "Scopes",
              group: "OAuth2 client",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1.5">
                  {config!.scopes.map((scope) => (
                    <Label key={scope} size="sm" outlined>
                      {scope}
                    </Label>
                  ))}
                </div>
              ),
            },
          ]
        : []),
    ],
  };
};
