import {
  getPB,
  getResourceClient,
  invalidateResource,
  invalidateResourceList,
  Resource,
  resourceMetadataToYAML,
  resourceSpecToJSON,
  resourceSpecToYAML,
  resourceStatusToJSON,
  resourceStatusToYAML,
  resourceToJSON,
  resourceToYAML,
} from "@/utils/pb";

import * as React from "react";

import Editor from "../Editor";

import { match } from "ts-pattern";

import { onError } from "@/utils";
import {
  Badge,
  Button,
  CopyButton,
  Drawer,
  Tooltip,
  Transition,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { FaCheckDouble } from "react-icons/fa6";
import { MdOutlineContentCopy } from "react-icons/md";
import { PiTextAlignLeftFill } from "react-icons/pi";
import { toast } from "sonner";

const ResourceYAML = (props: {
  item: Resource;
  size?: "xs" | "small";
  btnItem?: boolean;
  triggerComponent?: React.ReactNode;
  onResourceChange?: (arg: Resource) => void;
  readOnly?: boolean;
  mode?: "json" | "yaml" | undefined;
}) => {
  let [mode, setMode] = React.useState(0);
  const { item } = props;

  const [opened, { open, close }] = useDisclosure(false);
  const [cur, setCur] = React.useState<Resource>(item);

  let [isChanged, setIsChanged] = React.useState(false);

  React.useEffect(() => {
    //@ts-ignore
    const val = !getPB(item)[`${item.kind}_Spec`].equals(item.spec, cur.spec);
    setIsChanged(val);
  }, [cur]);

  const mutationUpdate = useMutation({
    mutationFn: async (req: Resource) => {
      // @ts-ignore
      const { response } =
        // @ts-ignore
        await getResourceClient(cur)[`update${req.kind}`](cur);
      return response as Resource;
    },

    onSuccess: (response) => {
      invalidateResource(response);
      invalidateResourceList(response);
      toast.success(
        `${response.kind}: ${response.metadata?.name} successfully updated`,
      );
    },
    onError,
  });

  const value = match(mode)
    .with(1, () =>
      props.mode === `json`
        ? resourceSpecToJSON(item)
        : resourceSpecToYAML(item),
    )
    .with(2, () =>
      props.mode === `json`
        ? resourceStatusToJSON(item)
        : resourceStatusToYAML(item),
    )
    .with(3, () => resourceMetadataToYAML(item))
    .otherwise(() =>
      props.mode === `json` ? resourceToJSON(item) : resourceToYAML(item),
    );

  return (
    <>
      <Button
        size={"compact-xs"}
        variant="outline"
        onClick={open}
        rightSection={<PiTextAlignLeftFill />}
        // className="font-bold transition-all duration-500 hover:text-black"
      >
        YAML
      </Button>
      <Drawer opened={opened} onClose={close} size={"lg"}>
        <div className="sm:max-w-sm md:max-w-[600px] h-full rounded-none">
          <div className="w-full py-4 px-2">
            <div className="flex flex-row items-center justify-center my-3">
              {/**
              <Group>
                <Select
                  label="Filter"
                  description="Whether you only want to show the Resource Spec, Status, Metadata or show all"
                  defaultValue={`${mode}`}
                  clearable
                  data={[
                    {
                      label: "Spec",
                      value: "1",
                    },
                    {
                      label: "Status",
                      value: "2",
                    },
                    {
                      label: "Metadata",
                      value: "3",
                    },
                  ]}
                  onChange={(v) => {
                    setMode(parseInt(v ?? "0"));
                  }}
                />
              </Group> 
               * **/}
            </div>
            <div>
              {!item.metadata?.isSystem && (
                <Transition
                  mounted={isChanged}
                  transition="fade"
                  duration={400}
                  timingFunction="ease"
                >
                  {(style) => (
                    <div
                      style={style}
                      className="flex my-4 items-center justify-end"
                    >
                      <Button
                        variant="filled"
                        size={"lg"}
                        loading={mutationUpdate.isPending}
                        onClick={() => {
                          mutationUpdate.mutate(cur);
                        }}
                      >
                        Update
                      </Button>
                    </div>
                  )}
                </Transition>
              )}

              <div className="w-full my-8 flex items-center">
                <CopyButton value={value}>
                  {({ copied, copy }) => (
                    <Button size="sm" className="shadow-lg" onClick={copy}>
                      <span className="mr-2">Copy Resource</span>
                      <AnimatePresence initial={false} mode="popLayout">
                        <motion.div
                          key={copied ? `1` : `2`}
                          initial={{ y: 30, opacity: 0 }}
                          animate={{ y: 0, opacity: 1 }}
                          exit={{ y: -30, opacity: 0 }}
                          transition={{ duration: 0.2, stiffness: 50 }}
                        >
                          {copied ? (
                            <FaCheckDouble />
                          ) : (
                            <MdOutlineContentCopy />
                          )}
                        </motion.div>
                      </AnimatePresence>
                    </Button>
                  )}
                </CopyButton>
                <div className="flex-1"></div>
                {item.metadata?.isSystem && (
                  <Tooltip
                    className="mr-2"
                    label={
                      <>This is a system Resource created by the Cluster</>
                    }
                  >
                    <Badge size="sm" variant="outline" color="blue">
                      System
                    </Badge>
                  </Tooltip>
                )}
              </div>

              <Editor
                item={props.item}
                mode={props.mode === `json` ? "json" : "yaml"}
                onResourceChange={(n) => {
                  setCur(n);
                }}
                value={value}
                readOnly={props.readOnly || item.metadata?.isSystem}
              />
            </div>
          </div>
        </div>
      </Drawer>
    </>
  );
};

export default ResourceYAML;
