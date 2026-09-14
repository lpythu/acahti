import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"

export function isMarkdownPath(name: string) {
  return /\.(md|mdx|markdown)$/i.test(name)
}

export function Markdown({ children }: { children: string }) {
  return (
    <div className="typeset typeset-docs max-w-[37em]">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{children}</ReactMarkdown>
    </div>
  )
}
