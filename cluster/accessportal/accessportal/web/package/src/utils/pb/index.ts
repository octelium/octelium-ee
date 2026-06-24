import * as AccessPB from "../../apis/accessv1/accessv1";
import * as MetaPB from "../../apis/metav1/metav1";

export type Resource = ResourceAccess;

export type ResourceAccess =
  | AccessPB.Policy
  | AccessPB.Request
  | AccessPB.Review
  | AccessPB.Catalog;

export const getResourceRef = (arg: Resource): MetaPB.ObjectReference =>
  MetaPB.ObjectReference.create({
    apiVersion: arg.apiVersion,
    kind: arg.kind,
    uid: arg.metadata?.uid,
    name: arg.metadata?.name,
  });
