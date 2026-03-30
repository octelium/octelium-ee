import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import { cloneResource } from "@/utils/pb";

const Edit = (props: {
  item: CoreP.Region;
  onUpdate: (item: CoreP.Region) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Region>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Region.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Region;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return <div></div>;
};

export default Edit;
