import { Certificate } from "@/apis/enterprisev1/enterprisev1";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button, CopyButton, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  CheckCheck,
  CheckCircle2,
  Copy,
  Loader2,
  ShieldCheck,
  Upload,
  X,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

export const SetCertificateC = (props: { item: Certificate }) => {
  const { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const [certificate, setCertificate] = React.useState("");
  const [privateKey, setPrivateKey] = React.useState("");

  const certFileRef = React.useRef<HTMLInputElement>(null);
  const keyFileRef = React.useRef<HTMLInputElement>(null);

  const readFile = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = (e) => resolve(e.target?.result as string);
      reader.onerror = reject;
      reader.readAsText(file);
    });

  const mutationSet = useMutation({
    mutationFn: async () => {
      const { response } = await getClientEnterprise().setCertificate({
        certificateRef: getResourceRef(item),
        certificate,
        privateKey,
      });
      return response;
    },
    onSuccess: () => {
      invalidateResource(item);
      invalidateResourceList(item);
      setCertificate("");
      setPrivateKey("");
      close();
      toast.success("Certificate set successfully");
    },
    onError,
  });

  const canSubmit =
    certificate.trim().length > 0 && privateKey.trim().length > 0;

  const PEMField = ({
    label,
    description,
    placeholder,
    value,
    onChange,
    fileRef,
  }: {
    label: string;
    description: string;
    placeholder: string;
    value: string;
    onChange: (v: string) => void;
    fileRef: React.RefObject<HTMLInputElement | null>;
  }) => (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-0.5">
          <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
            {label}
          </span>
          <span className="text-[0.68rem] font-semibold text-slate-400">
            {description}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          {value && (
            <CopyButton value={value}>
              {({ copied, copy }) => (
                <button
                  onClick={copy}
                  className="flex items-center gap-1 px-2 py-1 rounded text-[0.68rem] font-bold border border-slate-200 bg-white hover:bg-slate-50 transition-colors duration-150 cursor-pointer text-slate-500 hover:text-slate-700"
                >
                  <AnimatePresence mode="popLayout" initial={false}>
                    <motion.span
                      key={copied ? "check" : "copy"}
                      initial={{ y: 4, opacity: 0 }}
                      animate={{ y: 0, opacity: 1 }}
                      exit={{ y: -4, opacity: 0 }}
                      transition={{ duration: 0.1 }}
                      className="flex items-center gap-1"
                    >
                      {copied ? (
                        <>
                          <CheckCheck size={11} strokeWidth={2.5} />
                          Copied
                        </>
                      ) : (
                        <>
                          <Copy size={11} strokeWidth={2.5} />
                          Copy
                        </>
                      )}
                    </motion.span>
                  </AnimatePresence>
                </button>
              )}
            </CopyButton>
          )}
          <button
            onClick={() => fileRef.current?.click()}
            className="flex items-center gap-1 px-2 py-1 rounded text-[0.68rem] font-bold border border-slate-200 bg-white hover:bg-slate-50 transition-colors duration-150 cursor-pointer text-slate-500 hover:text-slate-700"
          >
            <Upload size={11} strokeWidth={2.5} />
            Upload file
          </button>
          <input
            ref={fileRef}
            type="file"
            accept=".pem,.crt,.cer,.key,.txt"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0];
              if (!file) return;
              const text = await readFile(file);
              onChange(text);
              e.target.value = "";
            }}
          />
        </div>
      </div>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={7}
        className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2.5 font-mono text-[0.72rem] text-slate-700 placeholder:text-slate-300 focus:border-slate-900 focus:outline-none focus:ring-0 resize-none transition-colors duration-500 shadow-[0_1px_3px_rgba(15,23,42,0.05)]"
        spellCheck={false}
      />
      {value && (
        <div className="flex items-center gap-1.5">
          <CheckCircle2
            size={11}
            strokeWidth={2.5}
            className="text-emerald-500 shrink-0"
          />
          <span className="text-[0.68rem] font-semibold text-emerald-600">
            {value.trim().split("\n").length} lines loaded
          </span>
          <button
            onClick={() => onChange("")}
            className="ml-auto flex items-center gap-1 text-[0.68rem] font-semibold text-slate-400 hover:text-red-500 transition-colors duration-150 cursor-pointer"
          >
            <X size={10} strokeWidth={2.5} />
            Clear
          </button>
        </div>
      )}
    </div>
  );

  return (
    <>
      <button
        onClick={open}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[0.75rem] font-bold border border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] text-slate-600"
      >
        <ShieldCheck size={13} strokeWidth={2.5} />
        Set Certificate
      </button>

      <Modal
        opened={opened}
        onClose={close}
        size="xl"
        centered
        withCloseButton={false}
        padding={0}
        styles={{
          content: {
            borderRadius: "12px",
            border: "1px solid #e2e8f0",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex flex-col">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 bg-slate-50/60">
            <div className="flex items-center gap-2">
              <ShieldCheck
                size={14}
                className="text-slate-500 shrink-0"
                strokeWidth={2}
              />
              <span className="text-[0.82rem] font-bold text-slate-800">
                Set Certificate
              </span>
              <span className="text-[0.7rem] font-semibold text-slate-400 font-mono">
                {item.metadata?.name}
              </span>
            </div>
            <button
              onClick={close}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors duration-150 cursor-pointer"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          </div>

          <div className="flex flex-col gap-5 px-5 py-5">
            <PEMField
              label="Certificate / Certificate Chain"
              description="PEM-encoded certificate or full chain (cert + intermediates)"
              placeholder={`-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU...\n-----END CERTIFICATE-----`}
              value={certificate}
              onChange={setCertificate}
              fileRef={certFileRef}
            />

            <PEMField
              label="Private Key"
              description="PEM-encoded private key corresponding to the certificate"
              placeholder={`-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0B...\n-----END PRIVATE KEY-----`}
              value={privateKey}
              onChange={setPrivateKey}
              fileRef={keyFileRef}
            />
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-slate-200 bg-slate-50/60">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              onClick={close}
              disabled={mutationSet.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="filled"
              color="dark"
              size="sm"
              disabled={!canSubmit || mutationSet.isPending}
              loading={mutationSet.isPending}
              leftSection={
                mutationSet.isPending ? (
                  <Loader2
                    size={13}
                    strokeWidth={2.5}
                    className="animate-spin"
                  />
                ) : (
                  <ShieldCheck size={13} strokeWidth={2.5} />
                )
              }
              onClick={() => mutationSet.mutate()}
            >
              {mutationSet.isPending ? "Setting…" : "Set Certificate"}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};
