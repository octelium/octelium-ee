import { Item } from "@/pages/visibility/Main";
import { SummaryItemCount, SummaryItemCountWrap, SummaryNoItems } from "@/components/Summary";
import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Summary as CertificateSummary } from "./Certificate/List";
import { Summary as CertificateIssuerSummary } from "./CertificateIssuer/List";
import { Summary as CollectorExporterSummary } from "./CollectorExporter/List";
import { Summary as DNSProviderSummary } from "./DNSProvider/List";
import { Summary as DirectoryProviderSummary } from "./DirectoryProvider/List";
import { Summary as SecretSummary } from "./Secret/List";
import { Summary as SecretStoreSummary } from "./SecretStore/List";

const DirectoryInventorySummary = () => {
  const users = useQuery({ queryKey: ["visibility", "enterprise", "summary", "DirectoryProviderUser"], queryFn: async () => (await getClientVisibilityEnterprise().getDirectoryProviderUserSummary({})).response });
  const groups = useQuery({ queryKey: ["visibility", "enterprise", "summary", "DirectoryProviderGroup"], queryFn: async () => (await getClientVisibilityEnterprise().getDirectoryProviderGroupSummary({})).response });
  if (!users.data && !groups.data) return null;
  if (!(users.data?.totalNumber || groups.data?.totalNumber)) return <SummaryNoItems />;
  return <SummaryItemCountWrap>
    <SummaryItemCount count={users.data?.totalNumber}>Directory users</SummaryItemCount>
    <SummaryItemCount count={users.data?.totalUser}>Linked users</SummaryItemCount>
    <SummaryItemCount count={groups.data?.totalNumber}>Directory groups</SummaryItemCount>
    <SummaryItemCount count={groups.data?.totalGroup}>Linked groups</SummaryItemCount>
    <SummaryItemCount count={Math.max(users.data?.totalDirectoryProvider ?? 0, groups.data?.totalDirectoryProvider ?? 0)}>Providers</SummaryItemCount>
  </SummaryItemCountWrap>;
};

export default () => (
  <motion.div
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, ease: "easeOut" }}
    className="grid grid-cols-1 gap-4 py-4 lg:grid-cols-2"
  >
    <Item title="Certificates" link="/enterprise/certificates"><CertificateSummary showNoItems /></Item>
    <Item title="Certificate Issuers" link="/enterprise/certificateissuers"><CertificateIssuerSummary showNoItems /></Item>
    <Item title="Directory Providers" link="/enterprise/directoryproviders"><DirectoryProviderSummary showNoItems /></Item>
    <Item title="Directory Inventory"><DirectoryInventorySummary /></Item>
    <Item title="Collector Exporters" link="/enterprise/collectorexporters"><CollectorExporterSummary showNoItems /></Item>
    <Item title="DNS Providers" link="/enterprise/dnsproviders"><DNSProviderSummary showNoItems /></Item>
    <Item title="Secret Stores" link="/enterprise/secretstores"><SecretStoreSummary showNoItems /></Item>
    <Item title="Secrets" link="/enterprise/secrets"><SecretSummary showNoItems /></Item>
  </motion.div>
);
