import * as CoreC from "@/apis/corev1/corev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { KeyRound, LaptopMinimal, Smartphone, Terminal } from "lucide-react";

export const useUserRelatedResources = (item: CoreC.User) => {
  const itemRef = getResourceRef(item);
  const itemName = item.metadata!.name;
  const queryKey = item.metadata!.uid || itemName;
  const common = CommonListOptions.create({ itemsPerPage: 1 });

  const query = useQuery({
    queryKey: ["core.user.relatedResources", queryKey],
    queryFn: async () => {
      const [sessions, devices, authenticators, credentials] =
        await Promise.all([
          getClientCore().listSession(
            CoreC.ListSessionOptions.create({ userRef: itemRef, common }),
          ).response,
          getClientCore().listDevice(
            CoreC.ListDeviceOptions.create({ userRef: itemRef, common }),
          ).response,
          getClientCore().listAuthenticator(
            CoreC.ListAuthenticatorOptions.create({
              userRef: itemRef,
              common,
            }),
          ).response,
          getClientCore().listCredential(
            CoreC.ListCredentialOptions.create({ userRef: itemRef, common }),
          ).response,
        ]);

      return {
        sessions: sessions.listResponseMeta?.totalCount,
        devices: devices.listResponseMeta?.totalCount,
        authenticators: authenticators.listResponseMeta?.totalCount,
        credentials: credentials.listResponseMeta?.totalCount,
      };
    },
  });

  return [
    {
      label: "Sessions",
      count: query.data?.sessions,
      path: `/core/sessions?userRef.name=${encodeURIComponent(itemName)}`,
      icon: Terminal,
    },
    {
      label: "Devices",
      count: query.data?.devices,
      path: `/core/devices?userRef.name=${encodeURIComponent(itemName)}`,
      icon: LaptopMinimal,
    },
    {
      label: "Authenticators",
      count: query.data?.authenticators,
      path: `/core/authenticators?userRef.name=${encodeURIComponent(itemName)}`,
      icon: Smartphone,
    },
    {
      label: "Credentials",
      count: query.data?.credentials,
      path: `/core/credentials?userRef.name=${encodeURIComponent(itemName)}`,
      icon: KeyRound,
    },
  ];
};
