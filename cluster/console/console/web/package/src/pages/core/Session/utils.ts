import { SessionExtInfo } from "@/apis/cordiumv1/cordiumv1";
import { Session } from "@/apis/corev1/corev1";
import { Struct } from "@/apis/google/protobuf/struct";

export const getCordiumSessionInfo = (
  item: Session,
): SessionExtInfo | undefined => {
  const ext = item.status?.ext?.cordium;
  if (!ext) return undefined;

  try {
    return SessionExtInfo.fromJson(Struct.toJson(ext), {
      ignoreUnknownFields: true,
    });
  } catch {
    return undefined;
  }
};
