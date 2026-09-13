import { useEffect } from "react"

export function useEvents(onEvent: () => void) {
  useEffect(() => {
    const es = new EventSource("/ui/events", { withCredentials: true })
    let timer = 0
    const bump = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(onEvent, 400)
    }
    es.onmessage = bump
    return () => {
      window.clearTimeout(timer)
      es.close()
    }
  }, [onEvent])
}
