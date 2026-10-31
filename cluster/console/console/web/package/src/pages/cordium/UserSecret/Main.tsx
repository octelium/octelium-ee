import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getUserSecretTypeLabel, userRefOf } from "../utils";
import { EyeOff, KeyRound, UserRound } from "lucide-react";

export const MainInfo = (props: {
  item: CordiumP.UserSecret;
}): ResourceMainInfo => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);
  const details = item.status?.details;
  const typeLabel = getUserSecretTypeLabel(item.spec?.type);

  return {
    status: { label: typeLabel, tone: "neutral" },
    groupOrder: [{ title: "Key", icon: KeyRound }],
    items: [
      {
        label: "Type",
        primary: true,
        icon: KeyRound,
        value: <Label>{typeLabel}</Label>,
      },
      ...(userRef
        ? [
            {
              label: "User",
              primary: true,
              icon: UserRound,
              value: <ResourceListLabel itemRef={userRef} />,
            },
          ]
        : []),
      ...(details?.oneofKind === "sshKey" && details.sshKey.publicKey
        ? [
            {
              label: "SSH public key",
              group: "Key",
              span: "full" as const,
              value: <CopyText value={details.sshKey.publicKey} />,
            },
          ]
        : []),
      {
        label: "Value",
        primary: true,
        icon: EyeOff,
        value: <Label>Write-only</Label>,
        hint: "The visibility API never returns the value.",
      },
    ],
  };
};
