import { Condition } from "@/apis/corev1/corev1";
import { Condition as Policy_Spec_Condition } from "@/apis/enterprisev1/enterprisev1";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { Button, Modal, ScrollArea } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import ConditionC from "./Condition";
import PrintCond from "./PrintCondition";

const ConditionBuilderBtn = (props: {
  onChange: (condition?: Condition) => void;
}) => {
  const [opened, { open, close }] = useDisclosure(false);

  let [cur, setCur] = useState<Policy_Spec_Condition | undefined>(undefined);

  const mutation = useMutation({
    mutationFn: async (req?: Policy_Spec_Condition) => {
      let r = await getClientEnterprise().getCoreCondition(
        req ?? Policy_Spec_Condition.create(),
      );

      return r.response;
    },
    onSuccess: (r) => {
      props.onChange(r);
    },
    onError,
  });

  return (
    <div>
      <Button
        size="xs"
        onClick={() => {
          open();
        }}
      >
        Condition UI Builder
      </Button>

      <Modal opened={opened} onClose={close} size={`100vw`}>
        <div>
          <div className="font-bold text-2xl text-shadow-2xs mb-8">
            Condition UI Builder
          </div>
          <div className="w-full my-8 flex">
            <div className="w-full flex-1">
              {cur && <PrintCond item={cur} />}
            </div>
            <div className="ml-4">
              <Button size="xl" onClick={close}>
                Done
              </Button>
            </div>
          </div>
          <div className="w-full">
            <ScrollArea h={500}>
              <div>
                <ConditionC
                  onChange={(v) => {
                    setCur(v);
                    mutation.mutate(v);
                  }}
                />
              </div>
            </ScrollArea>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default ConditionBuilderBtn;
