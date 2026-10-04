import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { spaceRefOf, userRefOf } from "../utils";

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
              value: <ResourceListLabel itemRef={spaceRef} />,
            },
          ]
        : []),
      ...(userRef
        ? [
            {
              label: "User",
              value: <ResourceListLabel itemRef={userRef} />,
              hint: "The User that created the Secret.",
            },
          ]
        : []),
      {
        label: "Data",
        value: (
          <span className="text-xs font-normal text-slate-500">
            The Secret data is never exposed by the visibility API.
          </span>
        ),
      },
    ],
  };
};
