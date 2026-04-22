import * as E from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getType } from "./List";

export default (props: { item: E.CertificateIssuer }) => {
  const { item } = props;
  return <></>;
};

export const MainInfo = (props: {
  item: E.CertificateIssuer;
}): ResourceMainInfo => {
  const { item } = props;
  return {
    items: [
      {
        label: "Type",
        value: (
          <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>
        ),
      },
    ],
  };
};
