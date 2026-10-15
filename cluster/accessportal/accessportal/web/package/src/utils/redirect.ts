export const getRedirectPath = (value: string | null, origin: string) => {
  if (!value || !value.startsWith("/") || value.startsWith("//")) return null;
  if (/[\\\u0000-\u0020\u007f]/.test(value)) return null;

  try {
    const url = new URL(value, origin);
    if (url.origin !== origin) return null;
    url.searchParams.delete("redirect");
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
};
