import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useT } from "@/i18n/i18n"

export function CopyField({ value, label, multiline }: { value: string; label?: string; multiline?: boolean }) {
  const t = useT()

  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(t("copied"))
    } catch {
      toast.error(t("copyFailed"))
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      {label ? <p className="text-sm text-muted-foreground">{label}</p> : null}
      <div className="flex gap-2">
        {multiline ? (
          <Textarea readOnly value={value} className="min-h-28 font-mono text-xs" />
        ) : (
          <Input readOnly value={value} className="font-mono text-xs" />
        )}
        <Button type="button" variant="outline" onClick={() => void copy()}>
          {t("copy")}
        </Button>
      </div>
    </div>
  )
}
