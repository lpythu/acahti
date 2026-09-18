import { Code2Icon } from "lucide-react"
import { toast } from "sonner"

import { MenuButton } from "@/components/menu-button"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useT } from "@/i18n/i18n"

export function CloneMenu({ url }: { url: string }) {
  const t = useT()

  async function copy() {
    try {
      await navigator.clipboard.writeText(url)
      toast.success(t("copied"))
    } catch {
      toast.error(t("copyFailed"))
    }
  }

  return (
    <MenuButton
      label={
        <>
          <Code2Icon />
          {t("clone")}
        </>
      }
      className="flex w-96 flex-col gap-3"
    >
      <div className="flex gap-2">
        <Input readOnly value={url} className="h-8 font-mono text-xs" />
        <Button type="button" size="sm" variant="outline" onClick={() => void copy()}>
          {t("copy")}
        </Button>
      </div>
    </MenuButton>
  )
}
