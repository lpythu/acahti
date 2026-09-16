import { type FormEvent, useMemo, useRef, useState } from "react"
import { toast } from "sonner"

import { MenuPanel } from "@/components/menu-panel"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import { api, type User } from "@/lib/api"
import { displayName } from "@/lib/user"

const PERMS = ["read", "write", "admin"] as const

export type AccessPerm = (typeof PERMS)[number]

const PERM_KEY: Record<AccessPerm, MessageKey> = {
  read: "permRead",
  write: "permWrite",
  admin: "permAdmin",
}

export function permLabel(t: (k: MessageKey) => string, perm: string) {
  const p = PERMS.includes(perm as AccessPerm) ? (perm as AccessPerm) : "read"
  return t(PERM_KEY[p])
}

export function PermSelect({
  value,
  onChange,
  disabled,
  className,
}: {
  value: string
  onChange: (v: AccessPerm) => void
  disabled?: boolean
  className?: string
}) {
  const t = useT()
  const current = PERMS.includes(value as AccessPerm) ? value : "read"
  return (
    <Select
      value={current}
      disabled={disabled}
      onValueChange={(v) => onChange((String(v ?? "read") as AccessPerm) || "read")}
      itemToStringLabel={(v) => permLabel(t, String(v ?? "read"))}
    >
      <SelectTrigger size="sm" className={className ?? "w-28"} disabled={disabled}>
        <SelectValue>{permLabel(t, current)}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        {PERMS.map((p) => (
          <SelectItem key={p} value={p}>
            {t(PERM_KEY[p])}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

async function loadUsers(): Promise<User[]> {
  const out: User[] = []
  for (let page = 1; page <= 20; page++) {
    const res = await api.users({ page, page_size: 50 })
    out.push(...res.items)
    if (!res.has_more) break
  }
  return out
}

export function AddPersonMenu({
  title,
  onAdd,
  exclude = [],
}: {
  title: string
  onAdd: (logins: string[], permission: AccessPerm) => Promise<void>
  exclude?: string[]
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [logins, setLogins] = useState<string[]>([])
  const [permission, setPermission] = useState<AccessPerm>("write")
  const [busy, setBusy] = useState(false)
  const inflight = useRef(false)
  const users = useLoad(() => loadUsers(), [], open)
  const taken = useMemo(() => new Set(exclude), [exclude])
  const options = useMemo(
    () =>
      (users.data ?? [])
        .filter((u) => u.login && !taken.has(u.login))
        .sort((a, b) => displayName(a).localeCompare(displayName(b), undefined, { sensitivity: "base" })),
    [users.data, taken],
  )
  const labels = useMemo(
    () => Object.fromEntries(options.map((u) => [u.login, displayName(u)])),
    [options],
  )

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!logins.length || inflight.current) return
    inflight.current = true
    setBusy(true)
    try {
      await onAdd(logins, permission)
      setLogins([])
      setPermission("write")
      setOpen(false)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("loadError"))
    } finally {
      inflight.current = false
      setBusy(false)
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setLogins([])
          setPermission("write")
          void users.reload()
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" />}>{title}</PopoverTrigger>
      <PopoverContent className="w-72">
        <MenuPanel loading={users.loading} error={users.error} empty={!users.loading && !options.length ? t("noUsers") : undefined}>
          <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel>{t("selectUser")}</FieldLabel>
              <Select
                multiple
                modal={false}
                value={logins}
                onValueChange={setLogins}
                items={labels}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("selectUser")} />
                </SelectTrigger>
                <SelectContent>
                  {options.map((u) => (
                    <SelectItem key={u.login} value={u.login}>
                      {displayName(u)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>{t("permission")}</FieldLabel>
              <PermSelect value={permission} onChange={setPermission} className="w-full" />
            </Field>
            <Button type="submit" disabled={!logins.length || busy}>
              {t("create")}
            </Button>
          </FieldGroup>
        </form>
        </MenuPanel>
      </PopoverContent>
    </Popover>
  )
}
