import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useT } from "@/i18n/i18n"

export function CopyField({ value, label }: { value: string; label?: string }) {
  const t = useT()
  const [done, setDone] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(value)
    setDone(true)
    window.setTimeout(() => setDone(false), 1500)
  }

  return (
    <div className="flex flex-col gap-1.5">
      {label ? <p className="text-sm text-muted-foreground">{label}</p> : null}
      <div className="flex gap-2">
        <Input readOnly value={value} className="font-mono text-xs" />
        <Button type="button" variant="outline" onClick={() => void copy()}>
          {done ? t("copied") : t("copy")}
        </Button>
      </div>
    </div>
  )
}
