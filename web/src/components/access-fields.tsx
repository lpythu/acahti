import { type FormEvent, useMemo, useState } from "react"

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
    <Select value={current} disabled={disabled} onValueChange={(v) => onChange((String(v ?? "read") as AccessPerm) || "read")}>
      <SelectTrigger size="sm" className={className ?? "w-28"} disabled={disabled}>
        <SelectValue />
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
  onAdd: (login: string, permission: AccessPerm) => Promise<void>
  exclude?: string[]
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [login, setLogin] = useState("")
  const [permission, setPermission] = useState<AccessPerm>("write")
  const [err, setErr] = useState("")
  const users = useLoad(() => loadUsers(), [], open)
  const taken = useMemo(() => new Set(exclude), [exclude])
  const options = useMemo(
    () =>
      (users.data ?? [])
        .filter((u) => u.login && !taken.has(u.login))
        .sort((a, b) => displayName(a).localeCompare(displayName(b), undefined, { sensitivity: "base" })),
    [users.data, taken],
  )

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!login) return
    setErr("")
    try {
      await onAdd(login, permission)
      setLogin("")
      setPermission("write")
      setOpen(false)
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("loadError"))
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setLogin("")
          setPermission("write")
          setErr("")
          void users.reload()
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" />}>{title}</PopoverTrigger>
      <PopoverContent className="w-72">
        <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel>{t("selectUser")}</FieldLabel>
              <Select value={login || null} onValueChange={(v) => setLogin(String(v ?? ""))}>
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
              {users.error ? <p className="text-sm text-destructive">{users.error}</p> : null}
              {!users.loading && !users.error && !options.length ? (
                <p className="text-sm text-muted-foreground">{t("noUsers")}</p>
              ) : null}
            </Field>
            <Field>
              <FieldLabel>{t("permission")}</FieldLabel>
              <PermSelect value={permission} onChange={setPermission} className="w-full" />
            </Field>
            {err ? <p className="text-sm text-destructive">{err}</p> : null}
            <Button type="submit" disabled={!login}>
              {t("create")}
            </Button>
          </FieldGroup>
        </form>
      </PopoverContent>
    </Popover>
  )
}
