import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import type { Components } from "react-markdown"

type Variant = "user" | "assistant"

function buildComponents(variant: Variant): Components {
  const isUser = variant === "user"
  const codeInline = isUser
    ? "rounded bg-primary-foreground/20 px-1 py-0.5 font-mono text-[0.9em]"
    : "rounded bg-black/10 px-1 py-0.5 font-mono text-[0.9em] dark:bg-white/10"
  const preWrap = isUser ? "bg-primary-foreground/15" : "bg-black/10 dark:bg-white/10"
  const linkClass = isUser
    ? "font-medium text-primary-foreground underline underline-offset-2"
    : "font-medium text-primary underline underline-offset-2"

  return {
    p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
    a: ({ href, children }) => (
      <a href={href} target="_blank" rel="noopener noreferrer" className={linkClass}>
        {children}
      </a>
    ),
    ul: ({ children }) => <ul className="my-2 list-disc space-y-1 pl-5">{children}</ul>,
    ol: ({ children }) => <ol className="my-2 list-decimal space-y-1 pl-5">{children}</ol>,
    li: ({ children }) => <li className="leading-relaxed">{children}</li>,
    h1: ({ children }) => (
      <h1 className="mb-2 mt-3 text-lg font-semibold first:mt-0">{children}</h1>
    ),
    h2: ({ children }) => (
      <h2 className="mb-2 mt-3 text-base font-semibold first:mt-0">{children}</h2>
    ),
    h3: ({ children }) => (
      <h3 className="mb-1 mt-2 text-sm font-semibold first:mt-0">{children}</h3>
    ),
    blockquote: ({ children }) => (
      <blockquote className="my-2 border-l-2 border-current/25 pl-3 italic opacity-95">
        {children}
      </blockquote>
    ),
    hr: () => <hr className="my-3 border-current/20" />,
    strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
    em: ({ children }) => <em>{children}</em>,
    code: ({ className, children, ...props }) => {
      const inline = !String(className ?? "").includes("language-")
      if (inline) {
        return (
          <code className={codeInline} {...props}>
            {children}
          </code>
        )
      }
      return (
        <code className={className} {...props}>
          {children}
        </code>
      )
    },
    pre: ({ children }) => (
      <pre
        className={`my-2 overflow-x-auto rounded-lg p-3 text-sm [&_code]:bg-transparent [&_code]:p-0 ${preWrap}`}
      >
        {children}
      </pre>
    ),
    table: ({ children }) => (
      <div className="my-2 overflow-x-auto">
        <table className="w-full border-collapse text-sm">{children}</table>
      </div>
    ),
    thead: ({ children }) => <thead>{children}</thead>,
    tbody: ({ children }) => <tbody>{children}</tbody>,
    tr: ({ children }) => <tr>{children}</tr>,
    th: ({ children }) => (
      <th className="border border-current/20 px-2 py-1 text-left font-semibold">{children}</th>
    ),
    td: ({ children }) => (
      <td className="border border-current/20 px-2 py-1 align-top">{children}</td>
    ),
  }
}

type Props = {
  content: string
  variant: Variant
}

/**
 * Renders chat text as Markdown (GFM). Plain text is valid — it becomes a paragraph.
 * No raw HTML (default react-markdown behavior).
 */
export function MessageMarkdown({ content, variant }: Props) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={buildComponents(variant)}>
      {content}
    </ReactMarkdown>
  )
}
