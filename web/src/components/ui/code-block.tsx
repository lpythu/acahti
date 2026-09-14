import { useEffect, useState } from "react"
import { cn } from "cn"

import { AutoHideScroll } from "@/components/ui/auto-hide-scroll"

export function CodeBlock({
  code,
  language,
  className,
}: {
  code: string
  language: string
  className?: string
}) {
  const [html, setHtml] = useState("")

  useEffect(() => {
    let live = true
    void import("shiki").then(async ({ codeToHtml }) => {
      const dark = document.documentElement.classList.contains("dark")
      try {
        const next = await codeToHtml(code, {
          lang: language,
          theme: dark ? "github-dark" : "github-light",
        })
        if (live) setHtml(next)
      } catch {
        const next = await codeToHtml(code, {
          lang: "text",
          theme: dark ? "github-dark" : "github-light",
        })
        if (live) setHtml(next)
      }
    })
    return () => {
      live = false
    }
  }, [code, language])

  return (
    <AutoHideScroll className={cn("size-full", className)}>
      {html ? (
        <div
          data-slot="code-block"
          className="[&_pre]:m-0 [&_pre]:bg-transparent! [&_pre]:p-4 [&_pre]:font-mono [&_pre]:text-xs [&_pre]:leading-relaxed"
          dangerouslySetInnerHTML={{ __html: html }}
        />
      ) : (
        <pre data-slot="code-block" className="m-0 p-4 font-mono text-xs leading-relaxed whitespace-pre">
          {code}
        </pre>
      )}
    </AutoHideScroll>
  )
}
