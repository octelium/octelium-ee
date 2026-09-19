import ResourceInventory from "@/components/ResourceInventory";
import Item from "@/components/SummaryCard";
import { motion } from "framer-motion";
import { Summary as AuthenticatorSummary } from "./Authenticator/List";
import { Summary as CredentialSummary } from "./Credential/List";
import { Summary as DeviceSummary } from "./Device/List";
import { Summary as GatewaySummary } from "./Gateway/List";
import { Summary as GroupSummary } from "./Group/List";
import { Summary as IdentityProviderSummary } from "./IdentityProvider/List";
import { Summary as NamespaceSummary } from "./Namespace/List";
import { Summary as PolicySummary } from "./Policy/List";
import { Summary as RegionSummary } from "./Region/List";
import { Summary as SecretSummary } from "./Secret/List";
import { Summary as ServiceSummary } from "./Service/List";
import { Summary as SessionSummary } from "./Session/List";
import { Summary as UserSummary } from "./User/List";

export default () => (
  <motion.div
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, ease: "easeOut" }}
    className="flex flex-col gap-4 py-4"
  >
    <ResourceInventory />

    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Item title="Services" link="/core/services">
        <ServiceSummary showNoItems />
      </Item>
      <Item title="Sessions" link="/core/sessions">
        <SessionSummary showNoItems />
      </Item>
      <Item title="Users" link="/core/users">
        <UserSummary showNoItems />
      </Item>
      <Item title="Devices" link="/core/devices">
        <DeviceSummary showNoItems />
      </Item>
      <Item title="Policies" link="/core/policies">
        <PolicySummary showNoItems />
      </Item>
      <Item title="Credentials" link="/core/credentials">
        <CredentialSummary showNoItems />
      </Item>
      <Item title="Identity Providers" link="/core/identityproviders">
        <IdentityProviderSummary showNoItems />
      </Item>
      <Item title="Authenticators" link="/core/authenticators">
        <AuthenticatorSummary showNoItems />
      </Item>
      <Item title="Namespaces" link="/core/namespaces">
        <NamespaceSummary />
      </Item>
      <Item title="Groups" link="/core/groups">
        <GroupSummary showNoItems />
      </Item>
      <Item title="Secrets" link="/core/secrets">
        <SecretSummary showNoItems />
      </Item>
      <Item title="Gateways" link="/core/gateways">
        <GatewaySummary />
      </Item>
      <Item title="Regions" link="/core/regions">
        <RegionSummary />
      </Item>
    </div>
  </motion.div>
);
