import {
  File,
  FileArchive,
  FileCode,
  FileImage,
  FileSpreadsheet,
  FileText,
  type LucideIcon,
} from "lucide-react";

export const fileIcon = (mimeType: string, name = ""): LucideIcon => {
  if (mimeType.startsWith("image/")) return FileImage;
  if (mimeType === "text/csv" || /\.(csv|tsv|xlsx?)$/i.test(name)) {
    return FileSpreadsheet;
  }
  if (/(zip|tar|gzip|compressed)/.test(mimeType)) return FileArchive;
  if (
    /(json|yaml|xml|javascript|typescript)/.test(mimeType) ||
    /\.(json|ya?ml|ts|js|go|py|sh|toml)$/i.test(name)
  ) {
    return FileCode;
  }
  if (mimeType.startsWith("text/") || mimeType === "application/pdf") {
    return FileText;
  }
  return File;
};

export const saveBlob = (blob: Blob, name: string) => {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 10000);
};

export const toFileName = (value: string, ext: string): string =>
  `${
    value
      .trim()
      .replace(/[^\w.-]+/g, "_")
      .slice(0, 80) || "conversation"
  }.${ext}`;
