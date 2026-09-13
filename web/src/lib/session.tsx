import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react"

import { api, type Me } from "@/lib/api"

const Session = createContext<{
  me: Me | null
  ready: boolean
  refresh: () => Promise<Me | null>
  clear: () => void
}>({
  me: null,
  ready: false,
  refresh: async () => null,
  clear: () => {},
})

export function SessionProvider({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me | null>(null)
  const [ready, setReady] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const next = await api.me()
      setMe(next)
      return next
    } catch {
      setMe(null)
      return null
    } finally {
      setReady(true)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <Session.Provider value={{ me, ready, refresh, clear: () => setMe(null) }}>
      {children}
    </Session.Provider>
  )
}

export function useSession() {
  return useContext(Session)
}
