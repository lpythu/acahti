import { type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"

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
}: {
  value: string
  onChange: (v: AccessPerm) => void
  disabled?: boolean
}) {
  const t = useT()
  const current = PERMS.includes(value as AccessPerm) ? value : "read"
  return (
    <Select value={current} disabled={disabled} onValueChange={(v) => onChange((String(v ?? "read") as AccessPerm) || "read")}>
      <SelectTrigger size="sm" className="w-28" disabled={disabled}>
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

export function AddPersonMenu({
  title,
  onAdd,
}: {
  title: string
  onAdd: (login: string, permission: AccessPerm) => Promise<void>
}) {
  const t = useT()
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const login = String(fd.get("login") || "").trim()
    const permission = (String(fd.get("permission") || "write") as AccessPerm) || "write"
    if (!login) return
    await onAdd(login, permission)
    e.currentTarget.reset()
  }
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" />}>{title}</PopoverTrigger>
      <PopoverContent className="w-72">
        <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="login">{t("username")}</FieldLabel>
              <Input id="login" name="login" required autoComplete="off" />
            </Field>
            <Field>
              <FieldLabel htmlFor="permission">{t("permission")}</FieldLabel>
              <select
                id="permission"
                name="permission"
                defaultValue="write"
                className="h-8 rounded-md border bg-background px-2 text-sm"
              >
                {PERMS.map((p) => (
                  <option key={p} value={p}>
                    {t(PERM_KEY[p])}
                  </option>
                ))}
              </select>
            </Field>
            <Button type="submit">{t("create")}</Button>
          </FieldGroup>
        </form>
      </PopoverContent>
    </Popover>
  )
}
