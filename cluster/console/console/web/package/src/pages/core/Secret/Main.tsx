import * as CoreC from "@/apis/corev1/corev1";
import Label from "@/components/Label";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { EyeOff, History } from "lucide-react";

export default (props: { item: CoreC.Secret }) => {
  const { item } = props;
  return <></>;
};

export const MainInfo = (props: { item: CoreC.Secret }): ResourceMainInfo => {
  const { item } = props;
  const md = item.metadata;

  return {
    items: [
      {
        label: "Value",
        primary: true,
        icon: EyeOff,
        value: <Label>Write-only</Label>,
        hint: "Only the Cluster reads the value; the API never returns it.",
      },
      {
        label: "Last changed",
        primary: true,
        icon: History,
        value: <TimeAgo rfc3339={md?.updatedAt ?? md?.createdAt} />,
      },
    ],
  };
};
