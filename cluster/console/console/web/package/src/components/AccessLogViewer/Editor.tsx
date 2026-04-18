import CodeMirror from "@uiw/react-codemirror";

import { json } from "@codemirror/lang-json";

import { AccessLog, ComponentLog } from "@/apis/corev1/corev1";
import { Button, CopyButton, Drawer } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";

import { AuditLog, AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { AnimatePresence, motion } from "framer-motion";
import { FaCheckDouble } from "react-icons/fa6";
import { MdOutlineContentCopy } from "react-icons/md";
import { match } from "ts-pattern";

const Editor = (props: {
  item: AccessLog | AuditLog | AuthenticationLog | ComponentLog;
}) => {
  let extensions = [json()];

  const [opened, { open, close }] = useDisclosure(false);
  const val = JSON.stringify(
    JSON.parse(
      match(props.item.kind)
        .with("AccessLog", () =>
          AccessLog.toJsonString(props.item as AccessLog),
        )
        .with("AuditLog", () => AuditLog.toJsonString(props.item as AuditLog))
        .with("AuthenticationLog", () =>
          AuthenticationLog.toJsonString(props.item as AuthenticationLog),
        )
        .with("ComponentLog", () =>
          ComponentLog.toJsonString(props.item as ComponentLog),
        )
        .otherwise(() => ""),
    ),
    null,
    2,
  );

  return (
    <>
      <Button size={"compact-xs"} variant="outline" onClick={open}>
        JSON
      </Button>

      <Drawer opened={opened} onClose={close} size={"lg"}>
        <div className="sm:max-w-sm md:max-w-[600px] h-full rounded-none">
          <div className="w-full py-4 px-2">
            <div className="font-bold text-sm text-slate-700 flex items-center my-8">
              <CopyButton value={val}>
                {({ copied, copy }) => (
                  <Button size="sm" className="shadow-lg" onClick={copy}>
                    <span className="mr-2">{`Copy ${props.item.kind}`}</span>
                    <AnimatePresence initial={false} mode="popLayout">
                      <motion.div
                        key={copied ? `1` : `2`}
                        initial={{ y: 30, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        exit={{ y: -30, opacity: 0 }}
                        transition={{ duration: 0.2, stiffness: 50 }}
                      >
                        {copied ? <FaCheckDouble /> : <MdOutlineContentCopy />}
                      </motion.div>
                    </AnimatePresence>
                  </Button>
                )}
              </CopyButton>
            </div>
            <div className="font-bold rounded-xl overflow-hidden w-full shadow-2xl text-xs">
              <CodeMirror
                value={val}
                autoFocus={true}
                readOnly={true}
                className="w-full"
                theme={"dark"}
                maxHeight="600px"
                minHeight="300px"
                extensions={extensions}
                // lang="json"
              />
            </div>
          </div>
        </div>
      </Drawer>
    </>
  );
};

export default Editor;
