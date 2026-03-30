import * as React from "react";

import { getClientCore } from "@/utils/client";

import * as CoreP from "@/apis/corev1/corev1";

import {
  cloneResource,
  getGetKeyFromPath,
  getListKeyFromPath,
} from "@/utils/pb";

const Edit = (props: {
  item: CoreP.Gateway;
  onUpdate: (item: CoreP.Gateway) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Gateway>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Gateway.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Gateway;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return <div></div>;
};

export default Edit;
