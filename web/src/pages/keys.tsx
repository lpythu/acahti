import { type FormEvent, useState } from "react"

import { PagedList } from "@/components/paged-list"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function KeysPage() {
  const t = useT()
  const list = usePage((q) => api.keys(q), [])
  const [formErr, setFormErr] = useState("")

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    setFormErr("")
    try {
      await api.addKey(String(fd.get("title") || ""), String(fd.get("key") || ""))
      e.currentTarget.reset()
      await list.reload()
    } catch (err) {
      setFormErr(err instanceof Error ? err.message : t("loadError"))
    }
  }

  return (
    <PagedList
      list={{ ...list, error: list.error || formErr }}
      header={
        <>
          <p className="text-sm text-muted-foreground">{t("keysDesc")}</p>
          <form className="max-w-lg" onSubmit={onSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="title">{t("title")}</FieldLabel>
                <Input id="title" name="title" defaultValue="laptop" required />
              </Field>
              <Field>
                <FieldLabel htmlFor="key">{t("key")}</FieldLabel>
                <Textarea id="key" name="key" rows={4} required placeholder="ssh-ed25519 AAAA…" />
              </Field>
              <Button type="submit">{t("addKey")}</Button>
            </FieldGroup>
          </form>
        </>
      }
      className="gap-6"
      skeleton="table"
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("title")}</TableHead>
              <TableHead>{t("key")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((k) => (
              <TableRow key={k.id}>
                <TableCell>{k.title}</TableCell>
                <TableCell className="max-w-md truncate font-mono text-xs">{k.key}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
