import { ObjectReference } from "@/apis/metav1/metav1";

export default (props: { sessionRef?: ObjectReference }) => {
  const { sessionRef } = props;
  if (!sessionRef) {
    return <></>;
  }
};
