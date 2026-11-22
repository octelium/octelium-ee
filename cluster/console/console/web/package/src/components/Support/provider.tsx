import { GetSupportInfoRequest } from "@/apis/enterprisev1/enterprisev1";
import { getClientSupport } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { SupportAvailabilityContext, supportProbeKey } from "./context";

export const SupportProvider = (props: { children: React.ReactNode }) => {
  const query = useQuery({
    queryKey: supportProbeKey,
    queryFn: async () => {
      const { response } = await getClientSupport().getSupportInfo(
        GetSupportInfoRequest.create({ skipPortal: true }),
      );
      return response;
    },
    retry: false,
    staleTime: 10 * 60_000,
    refetchOnWindowFocus: false,
  });

  return (
    <SupportAvailabilityContext.Provider value={query.isSuccess}>
      {props.children}
    </SupportAvailabilityContext.Provider>
  );
};
