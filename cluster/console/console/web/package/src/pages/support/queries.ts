import {
  AddSupportRequestAttachmentRequest,
  DownloadSupportRequestAttachmentRequest,
  GetSupportInfoRequest,
  GetSupportRequestRequest,
  ListSupportRequestOptions,
  ListSupportRequestOptions_Filter,
  SupportInfo_Limits,
  SupportRequest,
  SupportRequest_Attachment,
  SupportRequest_Attachment_Purpose,
} from "@/apis/enterprisev1/enterprisev1";
import { saveBlob } from "@/pages/Agent/files";
import { getClientSupport } from "@/utils/client";
import { retryQuery } from "@/utils/queryPolicy";
import { notifications } from "@mantine/notifications";
import { QueryClient, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { checkFile, fileKey, newIdempotencyKey } from "./utils";

export const supportKeys = {
  all: ["support"],
  info: ["support", "info"],
  requests: ["support", "requests"],
  list: (filter: ListSupportRequestOptions_Filter) => [
    "support",
    "requests",
    filter,
  ],
  request: (id: string) => ["support", "request", id],
};

export interface PendingFile {
  key: string;
  file: File;
  purpose: SupportRequest_Attachment_Purpose;
  idempotencyKey: string;
}

export const toPendingFile = (
  file: File,
  purpose = SupportRequest_Attachment_Purpose.FILE,
): PendingFile => ({
  key: `${purpose}:${fileKey(file)}`,
  file,
  purpose,
  idempotencyKey: newIdempotencyKey(),
});

export const admitFiles = (
  current: PendingFile[],
  items: PendingFile[],
  limits: SupportInfo_Limits | undefined,
  label: string,
) => {
  const max = limits?.maxAttachmentsPerMessage ?? 10;
  const errors: string[] = [];
  const next = [...current];
  for (const item of items) {
    const error = checkFile(limits, item.file);
    if (error) {
      errors.push(error);
      continue;
    }
    if (next.some((existing) => existing.key === item.key)) continue;
    if (next.length >= max) {
      errors.push(`${label} can include at most ${max} files`);
      break;
    }
    next.push(item);
  }
  for (const error of new Set(errors)) {
    notifications.show({ color: "red", message: error });
  }
  return next;
};

export const updateSupportRequest = (
  queryClient: QueryClient,
  id: string,
  response: SupportRequest,
) => {
  if (response.actions) {
    queryClient.setQueryData(supportKeys.request(id), response);
  } else {
    queryClient.invalidateQueries({ queryKey: supportKeys.request(id) });
  }
  queryClient.invalidateQueries({ queryKey: supportKeys.requests });
};

export const useSupportInfo = () =>
  useQuery({
    queryKey: supportKeys.info,
    queryFn: async () => {
      const { response } = await getClientSupport().getSupportInfo(
        GetSupportInfoRequest.create({}),
      );
      return response;
    },
    retry: (failureCount, error) =>
      failureCount < 1 && retryQuery(failureCount, error),
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  });

export const useSupportRequests = (
  filter: ListSupportRequestOptions_Filter,
  enabled: boolean,
) =>
  useInfiniteQuery({
    queryKey: supportKeys.list(filter),
    queryFn: async ({ pageParam }) => {
      const { response } = await getClientSupport().listSupportRequest(
        ListSupportRequestOptions.create({ filter, cursor: pageParam }),
      );
      return response;
    },
    initialPageParam: "",
    getNextPageParam: (page) => page.nextCursor || undefined,
    enabled,
    refetchInterval: 60_000,
    refetchOnWindowFocus: false,
  });

export const useSupportRequest = (id: string) =>
  useQuery({
    queryKey: supportKeys.request(id),
    queryFn: async () => {
      const { response } = await getClientSupport().getSupportRequest(
        GetSupportRequestRequest.create({ id }),
      );
      return response;
    },
    refetchInterval: 30_000,
    refetchOnWindowFocus: false,
  });

export const uploadFile = async (
  id: string,
  item: PendingFile,
): Promise<SupportRequest_Attachment> => {
  const data = new Uint8Array(await item.file.arrayBuffer());
  const { response } = await getClientSupport().addSupportRequestAttachment(
    AddSupportRequestAttachmentRequest.create({
      id,
      name: item.file.name,
      data,
      purpose: item.purpose,
      idempotencyKey: item.idempotencyKey,
    }),
  );
  return response;
};

export const uploadFiles = async (
  id: string,
  items: PendingFile[],
  uploaded: Map<string, string>,
  onProgress?: (done: number) => void,
) => {
  const ids: string[] = [];
  const failed: { item: PendingFile; error: unknown }[] = [];
  for (const [index, item] of items.entries()) {
    let attachmentID = uploaded.get(item.key);
    if (!attachmentID) {
      try {
        attachmentID = (await uploadFile(id, item)).id;
        uploaded.set(item.key, attachmentID);
      } catch (error) {
        failed.push({ item, error });
        continue;
      } finally {
        onProgress?.(index + 1);
      }
    }
    ids.push(attachmentID);
  }
  return { ids, failed };
};

export const downloadAttachment = async (
  request: SupportRequest,
  attachment: SupportRequest_Attachment,
) => {
  const call = getClientSupport().downloadSupportRequestAttachment(
    DownloadSupportRequestAttachmentRequest.create({
      id: request.id,
      attachmentID: attachment.id,
    }),
  );
  const chunks: Uint8Array<ArrayBuffer>[] = [];
  let name = attachment.name;
  for await (const message of call.responses) {
    if (message.type.oneofKind === "header") {
      name = message.type.header.name || name;
    } else if (message.type.oneofKind === "chunk") {
      chunks.push(new Uint8Array(message.type.chunk));
    }
  }
  const blob = new Blob(chunks, { type: "application/octet-stream" });
  if (attachment.size > 0 && blob.size !== Number(attachment.size)) {
    throw new Error("The file download was incomplete. Please try again");
  }
  if (attachment.sha256 && crypto.subtle) {
    const digest = new Uint8Array(
      await crypto.subtle.digest("SHA-256", await blob.arrayBuffer()),
    );
    const hex = Array.from(digest, (byte) =>
      byte.toString(16).padStart(2, "0"),
    ).join("");
    if (hex !== attachment.sha256.toLowerCase()) {
      throw new Error("The downloaded file is corrupted. Please try again");
    }
  }
  saveBlob(blob, name);
};
