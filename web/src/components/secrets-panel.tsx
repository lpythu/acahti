import { type FormEvent, useState } from "react"
import { toast } from "sonner"

import { Pager } from "@/components/paged-list"
import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import type { PipelineSecret } from "@/lib/api"
import type { Page, PageQuery } from "@/lib/page"

const DEFAULT_EVENTS = ["push", "tag", "manual"]

export function SecretsPanel({
  load,
  onPut,
  onDelete,
  deps = [],
  canDelete,
}: {
  load: (q: PageQuery) => Promise<Page<PipelineSecret>>
  onPut: (name: string, value: string, events: string[]) => Promise<unknown>
  onDelete: (name: string) => Promise<unknown>
  deps?: readonly unknown[]
  canDelete?: (s: PipelineSecret) => boolean
}) {
  const t = useT()
  const page = usePage(load, deps)
  const [name, setName] = useState("")
  const [value, setValue] = useState("")
  const [events, setEvents] = useState<string[]>(DEFAULT_EVENTS)

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const n = name.trim().toLowerCase()
    if (!n || !value) return
    try {
      await onPut(n, value, events)
      toast.success(t("secretSaved"))
      setName("")
      setValue("")
      setEvents(DEFAULT_EVENTS)
      await page.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    }
  }

  return (
    <PageFrame loading={page.loading && page.empty} error={page.error} className="gap-8">
      <form onSubmit={(e) => void submit(e)} className="flex max-w-xl flex-col gap-3">
        <h2 className="text-sm font-medium">{t("addSecret")}</h2>
        <p className="text-sm text-muted-foreground">{t("secretsHint")}</p>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="secret-name">{t("name")}</FieldLabel>
            <Input
              id="secret-name"
              name="secret-name"
              autoComplete="off"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="secret-value">{t("secretValue")}</FieldLabel>
            <Textarea
              id="secret-value"
              name="secret-value"
              autoComplete="off"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              required
              className="min-h-24 font-mono text-sm"
            />
          </Field>
          <Field>
            <FieldLabel>{t("secretEvents")}</FieldLabel>
            <div className="flex flex-wrap gap-3 text-sm">
              {DEFAULT_EVENTS.map((ev) => (
                <label key={ev} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={events.includes(ev)}
                    onChange={(e) => {
                      setEvents((cur) =>
                        e.target.checked ? [...cur, ev] : cur.filter((x) => x !== ev),
                      )
                    }}
                  />
                  {ev}
                </label>
              ))}
            </div>
          </Field>
          <Button type="submit">{t("saveSecret")}</Button>
        </FieldGroup>
      </form>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-medium">{t("tabSecrets")}</h2>
        {page.empty ? (
          <p className="text-sm text-muted-foreground">{t("noSecrets")}</p>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("name")}</TableHead>
                  <TableHead>{t("secretScope")}</TableHead>
                  <TableHead>{t("secretEvents")}</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {page.items.map((s) => (
                  <TableRow key={`${s.scope || "repo"}:${s.name}`}>
                    <TableCell className="font-mono text-sm">{s.name}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {s.scope === "org" ? t("secretScopeOrg") : t("secretScopeRepo")}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {(s.events || []).join(", ") || "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      {canDelete && !canDelete(s) ? null : (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            if (!window.confirm(t("deleteSecretConfirm", { name: s.name }))) return
                            void onDelete(s.name)
                              .then(async () => {
                                toast.success(t("secretDeleted"))
                                await page.reload()
                              })
                              .catch((err) => toast.error(err instanceof Error ? err.message : t("loadError")))
                          }}
                        >
                          {t("remove")}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pager page={page.page} hasMore={page.hasMore} onPage={page.setPage} />
          </>
        )}
      </section>
    </PageFrame>
  )
}
