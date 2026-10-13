export const normalize = (value: string) =>
  value.toLowerCase().normalize("NFKD").replace(/[̀-ͯ]/g, "").trim();

const isWordSubsequence = (query: string, word: string) => {
  if (word[0] !== query[0]) return false;
  let idx = 0;
  for (let pos = 0; pos < word.length; pos++) {
    if (word[pos] !== query[idx]) continue;
    idx++;
    if (idx === query.length) return pos < query.length * 1.5;
  }
  return false;
};

const scoreToken = (token: string, text: string): number => {
  if (text === token) return 100;
  if (text.startsWith(token)) return 80;
  const words = text.split(/[\s./_\-›·:]+/).filter(Boolean);
  if (words.some((word) => word.startsWith(token))) return 60;
  if (text.includes(token)) return 40;
  const initials = words.map((word) => word[0]).join("");
  if (token.length > 1 && initials.startsWith(token)) return 50;
  if (
    token.length > 3 &&
    words.some((word) => isWordSubsequence(token, word))
  ) {
    return 10;
  }
  return 0;
};

export const scoreText = (query: string, text: string): number => {
  const all = normalize(query).split(/\s+/).filter(Boolean);
  const long = all.filter((token) => token.length > 1);
  const tokens = long.length > 0 ? long : all;
  if (tokens.length === 0) return 0;
  const target = normalize(text);
  const whole = scoreToken(tokens.join(" "), target);
  if (whole > 0) return whole;
  let total = 0;
  for (const token of tokens) {
    const score = scoreToken(token, target);
    if (score === 0) return 0;
    total += score;
  }
  return (total / tokens.length) * 0.9;
};

export const scoreEntry = (
  query: string,
  entry: { title: string; subtitle?: string; keywords?: string[] },
): number =>
  Math.max(
    scoreText(query, entry.title),
    ...(entry.keywords ?? []).map((keyword) => scoreText(query, keyword) * 0.8),
    entry.subtitle
      ? scoreText(query, `${entry.subtitle} ${entry.title}`) * 0.7
      : 0,
  );

export const isQuestion = (query: string) => {
  const value = query.trim();
  return (
    value.endsWith("?") ||
    value.split(/\s+/).length >= 4 ||
    /^(how|why|what|which|who|when|where|explain|summarize|chart|is|are|can|does|do)\b/i.test(
      value,
    )
  );
};
