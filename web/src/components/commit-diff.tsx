import { useState } from "react"
import { ChevronDownIcon, ChevronRightIcon } from "lucide-react"
import { cn } from "cn"

import { useT } from "@/i18n/i18n"
import type { CommitFile } from "@/lib/api"
import { diffLines } from "@/lib/diff"

const ROW: Record<string, string> = {
  add: "bg-emerald-500/10 text-emerald-950 dark:text-emerald-100",
  del: "bg-red-500/10 text-red-950 dark:text-red-100",
  hunk: "bg-sky-500/10 text-sky-800 dark:text-sky-200",
  meta: "text-muted-foreground",
  ctx: "",
}

function FileDiff({ file }: { file: CommitFile }) {
  const [open, setOpen] = useState(true)
  const lines = diffLines(file.patch)
  const name =
    file.status === "renamed" && file.previous_filename
      ? `${file.previous_filename} → ${file.filename}`
      : file.filename

  return (
    <section className="overflow-hidden rounded-md border">
      <button
        type="button"
        className="flex w-full items-center gap-2 bg-muted/40 px-3 py-2 text-left text-sm"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? <ChevronDownIcon className="size-4" /> : <ChevronRightIcon className="size-4" />}
        <span className="min-w-0 flex-1 truncate font-mono">{name}</span>
        <span className="tabular-nums text-emerald-700 dark:text-emerald-400">+{file.additions}</span>
        <span className="tabular-nums text-red-700 dark:text-red-400">−{file.deletions}</span>
      </button>
      {open ? (
        lines.length ? (
          <div className="overflow-x-auto">
            <table className="w-full border-t font-mono text-xs">
              <tbody>
                {lines.map((l, i) => (
                  <tr key={i} className={cn("align-top", ROW[l.kind])}>
                    <td className="w-10 select-none px-2 py-0 text-right text-muted-foreground tabular-nums">
                      {l.oldNo || ""}
                    </td>
                    <td className="w-10 select-none px-2 py-0 text-right text-muted-foreground tabular-nums">
                      {l.newNo || ""}
                    </td>
                    <td className="whitespace-pre px-2 py-0">
                      <span className="mr-2 inline-block w-3 text-muted-foreground">
                        {l.kind === "add" ? "+" : l.kind === "del" ? "−" : l.kind === "hunk" ? "" : " "}
                      </span>
                      {l.text}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="border-t px-3 py-2 text-xs text-muted-foreground">binary</p>
        )
      ) : null}
    </section>
  )
}

export function CommitDiff({ files }: { files: CommitFile[] }) {
  const t = useT()
  if (!files.length) return <p className="text-sm text-muted-foreground">{t("noDiff")}</p>
  return (
    <div className="flex flex-col gap-3">
      {files.map((f) => (
        <FileDiff key={f.filename} file={f} />
      ))}
    </div>
  )
}
