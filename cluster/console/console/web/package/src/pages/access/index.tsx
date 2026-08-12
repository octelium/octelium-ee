import { Item } from "@/pages/visibility/Main";
import { motion } from "framer-motion";
import { Summary as CatalogSummary } from "./Catalog/List";
import { Summary as PolicySummary } from "./Policy/List";
import { Summary as RequestSummary } from "./Request/List";
import { Summary as ReviewSummary } from "./Review/List";

export default () => (
  <motion.div
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, ease: "easeOut" }}
    className="grid grid-cols-1 gap-4 py-4 lg:grid-cols-2"
  >
    <Item title="Access Requests" link="/access/requests"><RequestSummary showNoItems /></Item>
    <Item title="Reviews" link="/access/reviews"><ReviewSummary showNoItems /></Item>
    <Item title="Access Policies" link="/access/policies"><PolicySummary showNoItems /></Item>
    <Item title="Catalogs" link="/access/catalogs"><CatalogSummary showNoItems /></Item>
  </motion.div>
);
