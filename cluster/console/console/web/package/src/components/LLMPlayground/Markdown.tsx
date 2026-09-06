import { ActionIcon, Tooltip } from "@mantine/core";
import { Check, Copy } from "lucide-react";
import * as React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { twMerge } from "tailwind-merge";

const CodeBlock = (props: { language?: string; code: string }) => {
  const [copied, setCopied] = React.useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(props.code);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  return (
    <div className="group relative my-3 overflow-hidden rounded-xl border border-slate-800 bg-slate-900">
      <div className="flex items-center justify-between border-b border-slate-800 px-3 py-1.5">
        <span className="text-[0.62rem] font-bold uppercase tracking-[0.08em] text-slate-400">
          {props.language || "code"}
        </span>
        <Tooltip label={copied ? "Copied" : "Copy code"} withArrow>
          <ActionIcon
            size="sm"
            variant="subtle"
            color="gray"
            aria-label="Copy code"
            onClick={copy}
          >
            {copied ? (
              <Check size={13} className="text-emerald-400" />
            ) : (
              <Copy size={13} className="text-slate-400" />
            )}
          </ActionIcon>
        </Tooltip>
      </div>
      <pre className="overflow-x-auto px-3 py-3">
        <code className="font-mono text-[0.74rem] leading-6 text-slate-100">
          {props.code}
        </code>
      </pre>
    </div>
  );
};

const Markdown = (props: { children: string; className?: string }) => (
  <div
    className={twMerge(
      "min-w-0 text-[0.82rem] leading-6 text-slate-700",
      props.className,
    )}
  >
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => (
          <p className="my-2 first:mt-0 last:mb-0">{children}</p>
        ),
        strong: ({ children }) => (
          <strong className="font-bold text-slate-900">{children}</strong>
        ),
        em: ({ children }) => <em className="italic">{children}</em>,
        del: ({ children }) => (
          <del className="text-slate-400 line-through">{children}</del>
        ),
        h1: ({ children }) => (
          <h1 className="mb-2 mt-4 text-base font-bold text-slate-900 first:mt-0">
            {children}
          </h1>
        ),
        h2: ({ children }) => (
          <h2 className="mb-2 mt-4 text-sm font-bold text-slate-900 first:mt-0">
            {children}
          </h2>
        ),
        h3: ({ children }) => (
          <h3 className="mb-1.5 mt-3 text-[0.82rem] font-bold text-slate-800 first:mt-0">
            {children}
          </h3>
        ),
        ul: ({ children }) => (
          <ul className="my-2 list-disc space-y-1 pl-5">{children}</ul>
        ),
        ol: ({ children }) => (
          <ol className="my-2 list-decimal space-y-1 pl-5">{children}</ol>
        ),
        li: ({ children }) => <li className="pl-0.5">{children}</li>,
        blockquote: ({ children }) => (
          <blockquote className="my-3 border-l-2 border-slate-300 bg-slate-50 py-1 pl-3 text-slate-600">
            {children}
          </blockquote>
        ),
        a: ({ children, href }) => (
          <a
            href={href}
            target="_blank"
            rel="noreferrer noopener"
            className="font-semibold text-blue-600 underline decoration-blue-300 underline-offset-2 hover:text-blue-700"
          >
            {children}
          </a>
        ),
        hr: () => <hr className="my-4 border-slate-200" />,
        img: ({ src, alt }) =>
          typeof src === "string" ? (
            <img
              src={src}
              alt={alt ?? ""}
              loading="lazy"
              className="my-3 max-h-[420px] w-auto max-w-full rounded-xl border border-slate-200 bg-white object-contain shadow-sm"
            />
          ) : null,
        table: ({ children }) => (
          <div className="my-3 w-full overflow-x-auto rounded-xl border border-slate-200">
            <table className="w-full border-collapse text-[0.76rem]">
              {children}
            </table>
          </div>
        ),
        thead: ({ children }) => (
          <thead className="bg-slate-50 text-slate-600">{children}</thead>
        ),
        th: ({ children }) => (
          <th className="border-b border-slate-200 px-3 py-2 text-left font-bold">
            {children}
          </th>
        ),
        td: ({ children }) => (
          <td className="border-b border-slate-100 px-3 py-2 align-top">
            {children}
          </td>
        ),
        pre: ({ children }) => <>{children}</>,
        code: ({ className, children, ...rest }) => {
          const text = String(children ?? "").replace(/\n$/, "");
          const language = /language-(\w+)/.exec(className ?? "")?.[1];
          const isBlock = language !== undefined || text.includes("\n");

          if (isBlock) {
            return <CodeBlock language={language} code={text} />;
          }

          return (
            <code
              className="rounded border border-slate-200 bg-slate-100 px-1 py-0.5 font-mono text-[0.76rem] font-semibold text-slate-800"
              {...rest}
            >
              {text}
            </code>
          );
        },
      }}
    >
      {props.children}
    </ReactMarkdown>
  </div>
);

export default Markdown;
