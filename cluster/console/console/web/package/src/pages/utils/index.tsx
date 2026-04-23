export const toURLWithQry = (
  path: string,
  _params: Record<string, string> | string,
): string => {
  return `${path}?${new URLSearchParams(_params).toString()}`;
};
