import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import Label from "@/components/Label";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { EyeOff, History } from "lucide-react";

export default (props: { item: EnterpriseP.Secret }) => {
  const { item } = props;
  return <></>;
};

export const MainInfo = (props: { item: EnterpriseP.Secret }): ResourceMainInfo => {
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
