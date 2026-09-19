import * as AccessP from "@/apis/accessv1/accessv1";
import { match } from "ts-pattern";

export const getSourceLabel = (
  source?: AccessP.IntegrationIdentity_Status_Source,
): string =>
  match(source)
    .with(
      AccessP.IntegrationIdentity_Status_Source.EMAIL_DISCOVERY,
      () => "Email discovery",
    )
    .otherwise(() => "Unset");
