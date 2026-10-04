import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getUserSecretTypeLabel, userRefOf } from "../utils";

export const MainInfo = (props: {
  item: CordiumP.UserSecret;
}): ResourceMainInfo => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);
  const details = item.status?.details;
  const typeLabel = getUserSecretTypeLabel(item.spec?.type);

  return {
    status: { label: typeLabel, tone: "neutral" },
    items: [
      {
        label: "Type",
        primary: true,
        value: <Label>{typeLabel}</Label>,
      },
      ...(userRef
        ? [{ label: "User", value: <ResourceListLabel itemRef={userRef} /> }]
        : []),
      ...(details?.oneofKind === "sshKey" && details.sshKey.publicKey
        ? [
            {
              label: "SSH public key",
              span: "full" as const,
              value: <CopyText value={details.sshKey.publicKey} />,
            },
          ]
        : []),
      {
        label: "Data",
        value: (
          <span className="text-xs font-normal text-slate-500">
            The UserSecret data is never exposed by the visibility API.
          </span>
        ),
      },
    ],
  };
};
