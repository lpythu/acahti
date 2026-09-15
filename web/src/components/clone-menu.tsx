import { useState } from "react"
import { Code2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { useT } from "@/i18n/i18n"

export function CloneMenu({ url }: { url: string }) {
  const t = useT()
  const [done, setDone] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(url)
    setDone(true)
    window.setTimeout(() => setDone(false), 1500)
  }

  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" variant="default" size="sm" />}>
        <Code2Icon />
        {t("clone")}
      </PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
        <div className="flex gap-2">
          <Input readOnly value={url} className="h-8 font-mono text-xs" />
          <Button type="button" size="sm" variant="outline" onClick={() => void copy()}>
            {done ? t("copied") : t("copy")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
