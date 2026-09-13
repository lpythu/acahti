import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useT } from "@/i18n/i18n"

function CopyRow({ label, command }: { label: string; command: string }) {
  const t = useT()
  const [done, setDone] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(command)
    setDone(true)
    window.setTimeout(() => setDone(false), 1500)
  }

  return (
    <div className="flex flex-col gap-1.5 px-1.5 py-1">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <div className="flex gap-2">
        <Input readOnly value={command} className="h-8 font-mono text-xs" />
        <Button type="button" size="sm" variant="outline" onClick={() => void copy()}>
          {done ? t("copied") : t("copy")}
        </Button>
      </div>
    </div>
  )
}

export function CloneMenu({ https, ssh }: { https: string; ssh: string }) {
  const t = useT()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" size="sm" />}>{t("clone")}</DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-96 min-w-80 p-2">
        <DropdownMenuLabel>{t("cloneHttps")}</DropdownMenuLabel>
        <CopyRow label={t("https")} command={`git clone ${https}`} />
        <DropdownMenuLabel>{t("cloneSsh")}</DropdownMenuLabel>
        <CopyRow label={t("ssh")} command={`git clone ${ssh}`} />
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
