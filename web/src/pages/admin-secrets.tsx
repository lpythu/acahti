import { SecretsPanel } from "@/components/secrets-panel"
import { api } from "@/lib/api"

export function AdminSecretsPage() {
  return (
    <SecretsPanel
      load={(q) => api.orgSecrets(q)}
      onPut={(name, value) => api.putOrgSecret(name, value)}
      onDelete={(name) => api.deleteOrgSecret(name)}
    />
  )
}
