export type DiffLine = {
  kind: "hunk" | "ctx" | "add" | "del" | "meta"
  text: string
  oldNo?: number
  newNo?: number
}

export function diffLines(patch?: string): DiffLine[] {
  if (!patch) return []
  const out: DiffLine[] = []
  let oldNo = 0
  let newNo = 0
  for (const raw of patch.split("\n")) {
    const hunk = raw.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
    if (hunk) {
      oldNo = Number(hunk[1])
      newNo = Number(hunk[2])
      out.push({ kind: "hunk", text: raw })
      continue
    }
    if (raw.startsWith("+")) {
      out.push({ kind: "add", text: raw.slice(1), newNo })
      newNo += 1
      continue
    }
    if (raw.startsWith("-")) {
      out.push({ kind: "del", text: raw.slice(1), oldNo })
      oldNo += 1
      continue
    }
    if (raw.startsWith("\\")) {
      out.push({ kind: "meta", text: raw })
      continue
    }
    const text = raw.startsWith(" ") ? raw.slice(1) : raw
    out.push({ kind: "ctx", text, oldNo, newNo })
    oldNo += 1
    newNo += 1
  }
  return out
}
