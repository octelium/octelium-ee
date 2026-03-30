import * as CoreC from "../../apis/corev1/corev1";

export const getServiceHostname = (arg: CoreC.Service): string => {
  if (arg.status!.namespaceRef?.name === `default`) {
    return arg.metadata!.name.split(".")[0];
  }
  return `${arg.metadata!.name}`;
};

export const getServicePrivateFQDN = (
  arg: CoreC.Service,
  domain: string
): string => {
  return `${getServiceHostname(arg)}.local.${domain}`;
};

export const getServicePublicFQDN = (
  arg: CoreC.Service,
  domain: string
): string => {
  return `${getServiceHostname(arg)}.${domain}`;
};

export const getServicePublicURL = (
  arg: CoreC.Service,
  domain: string
): string => {
  return `https://${getServicePublicFQDN(arg, domain)}`;
};
