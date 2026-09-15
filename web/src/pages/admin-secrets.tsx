import { SecretsPanel } from "@/components/secrets-panel"
import { api } from "@/lib/api"

export function AdminSecretsPage() {
  return (
    <SecretsPanel
      load={(q) => api.orgSecrets(q)}
      onPut={(name, value, events) => api.putOrgSecret(name, value, events)}
      onDelete={(name) => api.deleteOrgSecret(name)}
    />
  )
}
