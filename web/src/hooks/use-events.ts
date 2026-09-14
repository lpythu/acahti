import { useEffect, useRef } from "react"

export type HubEvent = {
  type: string
  at: number
  data?: unknown
}

export function useEvents(onEvent: (ev: HubEvent) => void) {
  const ref = useRef(onEvent)
  ref.current = onEvent
  useEffect(() => {
    const es = new EventSource("/ui/events", { withCredentials: true })
    es.onmessage = (e) => {
      try {
        ref.current(JSON.parse(e.data) as HubEvent)
      } catch {
        /* ignore */
      }
    }
    return () => es.close()
  }, [])
}
