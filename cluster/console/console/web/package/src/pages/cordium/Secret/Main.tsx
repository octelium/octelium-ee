import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { spaceRefOf, userRefOf } from "../utils";
import Label from "@/components/Label";
import { Boxes, EyeOff, UserRound } from "lucide-react";

export const MainInfo = (props: {
  item: CordiumP.Secret;
}): ResourceMainInfo => {
  const { item } = props;
  const userRef = userRefOf(item.status?.userRef);
  const spaceRef = spaceRefOf(item.status?.spaceRef);

  return {
    items: [
      ...(spaceRef
        ? [
            {
              label: "Space",
              primary: true,
              icon: Boxes,
              value: <ResourceListLabel itemRef={spaceRef} />,
            },
          ]
        : []),
      ...(userRef
        ? [
            {
              label: "Created by",
              primary: true,
              icon: UserRound,
              value: <ResourceListLabel itemRef={userRef} />,
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
