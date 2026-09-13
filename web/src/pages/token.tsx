import { useState } from "react"

import { PageFrame } from "@/components/page-frame"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function TokenPage() {
  const t = useT()
  const { data, error, loading, reload } = useLoad(async () => (await api.tokens()).tokens || [], [])
  const [token, setToken] = useState("")
  const [actionErr, setActionErr] = useState("")
  const tokens = data || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error || actionErr}
      empty={!!data && tokens.length === 0 && !token}
      emptyText={t("noTokens")}
      className="max-w-3xl gap-6"
      skeleton="table"
      header={
        <>
          <p className="text-sm text-muted-foreground">{t("tokenDesc")}</p>
          {token ? (
            <Alert>
              <AlertTitle>{t("copyNow")}</AlertTitle>
              <AlertDescription>
                <code className="block break-all text-sm">{token}</code>
              </AlertDescription>
            </Alert>
          ) : null}
          <Button
            onClick={async () => {
              setActionErr("")
              try {
                const r = await api.issueToken()
                setToken(r.token)
                await reload()
              } catch (err) {
                setActionErr(err instanceof Error ? err.message : t("loadError"))
              }
            }}
          >
            {t("issueToken")}
          </Button>
          <div>
            <h2 className="text-sm font-medium">{t("tokens")}</h2>
            <p className="text-sm text-muted-foreground">{t("tokenListDesc")}</p>
          </div>
        </>
      }
    >
      {tokens.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("name")}</TableHead>
              <TableHead>{t("lastEight")}</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {tokens.map((tok) => (
              <TableRow key={tok.id}>
                <TableCell>{tok.name}</TableCell>
                <TableCell className="font-mono text-xs">{tok.token_last_eight}</TableCell>
                <TableCell>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={async () => {
                      await api.revokeToken(tok.id)
                      if (token.endsWith(tok.token_last_eight || "")) setToken("")
                      await reload()
                    }}
                  >
                    {t("revoke")}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
