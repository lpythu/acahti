import { type FormEvent, useState } from "react"

import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function KeysPage() {
  const t = useT()
  const { data, error, loading, reload } = useLoad(async () => (await api.keys()).keys || [], [])
  const [formErr, setFormErr] = useState("")
  const keys = data || []

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    setFormErr("")
    try {
      await api.addKey(String(fd.get("title") || ""), String(fd.get("key") || ""))
      e.currentTarget.reset()
      await reload()
    } catch (err) {
      setFormErr(err instanceof Error ? err.message : t("loadError"))
    }
  }

  return (
    <PageFrame
      loading={loading && !data}
      error={error || formErr}
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
      {keys.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("title")}</TableHead>
              <TableHead>{t("key")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((k) => (
              <TableRow key={k.id}>
                <TableCell>{k.title}</TableCell>
                <TableCell className="max-w-md truncate font-mono text-xs">{k.key}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
