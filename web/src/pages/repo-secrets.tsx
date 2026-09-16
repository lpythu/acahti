import { SecretsPanel } from "@/components/secrets-panel"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoSecretsPage() {
  const { owner, name } = useRepo()
  return (
    <SecretsPanel
      load={(q) => api.repoSecrets(owner, name, q)}
      onPut={(secret, value, events) => api.putRepoSecret(owner, name, secret, value, events)}
      onDelete={(secret) => api.deleteRepoSecret(owner, name, secret)}
      deps={[owner, name]}
      canDelete={(s) => s.scope !== "org"}
    />
  )
}
