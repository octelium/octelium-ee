import ClusterConfig from "./enterprise/ClusterConfig.json";
import Secret from "./enterprise/Secret.json";
import Certificate from "./enterprise/Certificate.json";
import CertificateIssuer from "./enterprise/CertificateIssuer.json";
import CollectorExporter from "./enterprise/CollectorExporter.json";
import DNSProvider from "./enterprise/DNSProvider.json";
import SecretStore from "./enterprise/SecretStore.json";
import DirectoryProvider from "./enterprise/DirectoryProvider.json";

import { ResourceEnterpriseName } from "@/utils/pb";
import { match } from "ts-pattern";

export default (arg: ResourceEnterpriseName) => {
  return match(arg)
    .with("ClusterConfig", () => ClusterConfig)
    .with("Secret", () => Secret)
    .with("Certificate", () => Certificate)
    .with("CertificateIssuer", () => CertificateIssuer)
    .with("CollectorExporter", () => CollectorExporter)
    .with("DNSProvider", () => DNSProvider)
    .with("SecretStore", () => SecretStore)
    .with("DirectoryProvider", () => DirectoryProvider)
    .otherwise(() => undefined);
};
