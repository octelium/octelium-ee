import {
  AddSupportRequestAttachmentRequest,
  DownloadSupportRequestAttachmentRequest,
  GetSupportInfoRequest,
  GetSupportRequestRequest,
  ListSupportRequestOptions,
  ListSupportRequestOptions_Filter,
  SupportRequest,
  SupportRequest_Attachment,
  SupportRequest_Attachment_Purpose,
} from "@/apis/enterprisev1/enterprisev1";
import { saveBlob } from "@/pages/Agent/files";
import { getClientSupport } from "@/utils/client";
import { retryQuery } from "@/utils/queryPolicy";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { fileKey } from "./utils";

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
}

export const toPendingFile = (
  file: File,
  purpose = SupportRequest_Attachment_Purpose.FILE,
): PendingFile => ({
  key: `${purpose}:${fileKey(file)}`,
  file,
  purpose,
});

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
  saveBlob(new Blob(chunks, { type: "application/octet-stream" }), name);
};
