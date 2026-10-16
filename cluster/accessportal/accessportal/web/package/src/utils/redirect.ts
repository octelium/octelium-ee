export const getRedirectPath = (value: string | null, origin: string) => {
  if (!value || !value.startsWith("/") || value.startsWith("//")) return null;
  if ([...value].some((char) => char === "\\" || char <= " " || char === "\u007f")) {
    return null;
  }

  try {
    const url = new URL(value, origin);
    if (url.origin !== origin) return null;
    url.searchParams.delete("redirect");
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
};
