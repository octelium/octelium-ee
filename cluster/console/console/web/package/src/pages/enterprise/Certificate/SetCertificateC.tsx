import { Certificate } from "@/apis/enterprisev1/enterprisev1";
import SecretTextAreaCustom from "@/components/TextAreaCustom/SecretTextAreaCustom";
import TextAreaCustom from "@/components/TextAreaCustom";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Alert, Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle, ShieldCheck, Upload, X } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

const isCertificatePEM = (value: string) =>
  value.includes("-----BEGIN CERTIFICATE-----") &&
  value.includes("-----END CERTIFICATE-----");

const isPrivateKeyPEM = (value: string) =>
  /-----BEGIN (?:RSA |EC |ENCRYPTED )?PRIVATE KEY-----/.test(value) &&
  /-----END (?:RSA |EC |ENCRYPTED )?PRIVATE KEY-----/.test(value);

export const SetCertificateC = (props: { item: Certificate }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [certificate, setCertificate] = React.useState("");
  const [privateKey, setPrivateKey] = React.useState("");

  const clearAndClose = () => {
    setCertificate("");
    setPrivateKey("");
    close();
  };

  const mutationSet = useMutation({
    mutationFn: async () => {
      const { response } = await getClientEnterprise().setCertificate({
        certificateRef: getResourceRef(props.item),
        certificate,
        privateKey,
      });
      return response;
    },
    onSuccess: () => {
      invalidateResource(props.item);
      invalidateResourceList(props.item);
      clearAndClose();
      toast.success("Certificate set successfully");
    },
    onError,
  });

  const certificateValid = isCertificatePEM(certificate.trim());
  const privateKeyValid = isPrivateKeyPEM(privateKey.trim());
  const canSubmit = certificateValid && privateKeyValid;

  return (
    <>
      <Button
        type="button"
        variant="default"
        size="compact-sm"
        leftSection={<Upload size={13} strokeWidth={2.4} />}
        onClick={open}
      >
        Set certificate
      </Button>

      <Modal
        opened={opened}
        onClose={clearAndClose}
        size="xl"
        centered
        withCloseButton={false}
        padding={0}
        styles={{
          content: {
            borderRadius: 14,
            border: "1px solid #e2e8f0",
            overflow: "hidden",
            maxHeight: "92vh",
          },
          body: { maxHeight: "92vh", overflow: "hidden" },
        }}
      >
        <div className="flex max-h-[92vh] flex-col bg-slate-50/40">
          <header className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 bg-white px-5 py-4">
            <div className="flex min-w-0 items-start gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <ShieldCheck size={17} strokeWidth={2.2} />
              </span>
              <div className="min-w-0">
                <h2 className="text-[0.86rem] font-bold text-slate-900">
                  Set certificate
                </h2>
                <p className="mt-0.5 truncate text-[0.69rem] font-semibold text-slate-400">
                  {props.item.metadata?.displayName ||
                    props.item.metadata?.name}
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="subtle"
              color="gray"
              size="compact-xs"
              onClick={clearAndClose}
            >
              <X size={14} />
            </Button>
          </header>

          <div className="flex-1 space-y-5 overflow-y-auto px-5 py-5">
            <Alert
              color="amber"
              icon={<AlertTriangle size={15} />}
              title="This replaces the certificate currently served"
            >
              Confirm that the certificate chain and private key belong
              together before applying them to the cluster.
            </Alert>

            <TextAreaCustom
              required
              rows={7}
              label="Certificate or certificate chain"
              description="PEM-encoded leaf certificate followed by any intermediate certificates."
              placeholder={`-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU...\n-----END CERTIFICATE-----`}
              value={certificate}
              onChange={(value) => setCertificate(value ?? "")}
            />
            {certificate && !certificateValid && (
              <p className="text-[0.68rem] font-semibold text-red-600">
                The value does not contain a complete PEM certificate block.
              </p>
            )}

            <SecretTextAreaCustom
              required
              rows={7}
              label="Private key"
              description="PEM-encoded private key corresponding to the leaf certificate."
              placeholder={`-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0B...\n-----END PRIVATE KEY-----`}
              value={privateKey}
              onChange={(value) => setPrivateKey(value ?? "")}
            />
            {privateKey && !privateKeyValid && (
              <p className="text-[0.68rem] font-semibold text-red-600">
                The value does not contain a complete PEM private-key block.
              </p>
            )}
          </div>

          <footer className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-5 py-3.5">
            <span className="text-[0.67rem] font-semibold text-slate-400">
              Sensitive input is cleared when this dialog closes.
            </span>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="default"
                size="sm"
                disabled={mutationSet.isPending}
                onClick={clearAndClose}
              >
                Cancel
              </Button>
              <Button
                type="button"
                color="dark"
                size="sm"
                disabled={!canSubmit}
                loading={mutationSet.isPending}
                leftSection={<ShieldCheck size={13} />}
                onClick={() => mutationSet.mutate()}
              >
                Apply certificate
              </Button>
            </div>
          </footer>
        </div>
      </Modal>
    </>
  );
};
