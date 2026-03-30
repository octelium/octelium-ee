import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";

import { useResourceForm } from "@/pages/utils/form";
import { Switch, Tabs } from "@mantine/core";
import { match } from "ts-pattern";

const Edit = (props: {
  item: EnterpriseP.DirectoryProvider;
  onUpdate: (item: EnterpriseP.DirectoryProvider) => void;
}) => {
  const { item, onUpdate } = props;

  const form = useResourceForm({ item, onUpdate });
  let cur = form.getValues();

  return (
    <div>
      <Switch
        label="Is Disabled"
        description="Disable the DirectoryProvider. Disabled DirectoryProviders cannot synchronize Users and Groups"
        key={form.key("spec.isDisabled")}
        {...form.getInputProps("spec.isDisabled")}
      />
      <Tabs
        defaultValue={match(cur?.spec)
          .when(
            (v) => v?.scim,
            () => "scim",
          )

          .otherwise(() => undefined)}
        onChange={(v) => {
          match(v).with("scim", () => {
            if (!cur?.spec?.scim) {
              form.setFieldValue("spec.scim", {});
            }
          });
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="scim">SCIM</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="scim">
          <></>
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};

export default Edit;
