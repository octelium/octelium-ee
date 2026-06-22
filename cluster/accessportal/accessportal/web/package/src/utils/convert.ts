export const strToNum = (input: string | number): number => {
  let result: number;

  if (typeof input === "number") {
    result = input;
  } else if (typeof input === "string") {
    const converted = +input;

    if (!isNaN(converted) && isFinite(converted)) {
      result = converted;
    } else {
      result = 0;
    }
  } else {
    result = 0;
  }

  return result;
};
