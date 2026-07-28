const DEFAULT_API_BASE = "/api"

export interface ClientRuntimeConfig {
  apiBase: string
}

let runtimeConfigPromise: Promise<ClientRuntimeConfig> | null = null

/**
 * Browser runtime config: from `config.json` in production (Docker entrypoint),
 * or same-origin `/api` in local Vite dev (no config.json).
 */
export async function getRuntimeConfig(): Promise<ClientRuntimeConfig> {
  if (runtimeConfigPromise) return runtimeConfigPromise
  runtimeConfigPromise = (async () => {
    if (import.meta.env.DEV) {
      return { apiBase: DEFAULT_API_BASE }
    }
    try {
      const res = await fetch("/config.json", { cache: "no-store" })
      if (res.ok) {
        const c = (await res.json()) as { apiBase?: string }
        return {
          apiBase: typeof c.apiBase === "string" && c.apiBase ? c.apiBase : DEFAULT_API_BASE,
        }
      }
    } catch {
      /* ignore */
    }
    return { apiBase: DEFAULT_API_BASE }
  })()
  return runtimeConfigPromise
}

async function getApiBase(): Promise<string> {
  const c = await getRuntimeConfig()
  return c.apiBase
}

async function parseJson(res: Response): Promise<unknown> {
  const text = await res.text()
  const ct = res.headers.get("content-type") ?? ""
  if (!ct.includes("application/json")) {
    throw new Error(
      `API returned ${res.status} (expected JSON, got ${ct || "unknown"}). ` +
        (text.startsWith("<") ? "Backend may be down or returning HTML." : "")
    )
  }
  try {
    return JSON.parse(text)
  } catch {
    throw new Error(`API returned invalid JSON`)
  }
}

export type ConversationStatus = "idle" | "running" | "completed" | "failed"

export interface Conversation {
  id: string
  title: string
  status?: ConversationStatus
  createdAt?: string
}

export interface Message {
  id: string
  role: "user" | "assistant"
  content: string
  createdAt?: string
}

export async function getConversations(): Promise<Conversation[]> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations`)
  if (!res.ok) throw new Error(`Failed to fetch chats: ${res.status}`)
  const data = (await parseJson(res)) as Conversation[] | { conversations?: Conversation[] }
  return Array.isArray(data) ? data : data.conversations ?? []
}

export async function createConversation(title = "New chat"): Promise<Conversation> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error(`Failed to create chat: ${res.status}`)
  const data = (await parseJson(res)) as { conversation?: Conversation } & Conversation
  const conv = data.conversation ?? data
  if (!conv?.id || !conv?.title) throw new Error("Invalid chat response")
  return conv
}

export async function getMessages(conversationId: string): Promise<Message[]> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}/messages`)
  if (!res.ok) throw new Error(`Failed to fetch messages: ${res.status}`)
  const data = (await parseJson(res)) as Message[] | { messages?: Message[] }
  return Array.isArray(data) ? data : data.messages ?? []
}

export async function renameConversation(
  conversationId: string,
  title: string
): Promise<void> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error(`Failed to rename chat: ${res.status}`)
}

export async function deleteConversation(conversationId: string): Promise<void> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(`Failed to delete chat: ${res.status}`)
}

// ── Streaming (AG-UI wire + server extension) ────────────────────────────────

/** AG-UI / SDK event discriminator strings from the stream. */
export const StreamEventType = {
  TEXT_MESSAGE_CONTENT: "TEXT_MESSAGE_CONTENT",
  RUN_ERROR: "RUN_ERROR",
  RUN_FINISHED: "RUN_FINISHED",
  /** App extension after RUN_FINISHED with persisted DB row (replaces legacy `done`). */
  MESSAGE_PERSISTED: "MESSAGE_PERSISTED",
} as const

export type StreamEvent =
  | { type: "TEXT_MESSAGE_CONTENT"; messageId?: string; delta: string; timestamp?: number }
  | { type: "RUN_ERROR"; message: string; code?: string; timestamp?: number }
  | { type: "RUN_FINISHED"; threadId?: string; runId?: string; result?: unknown; timestamp?: number }
  | { type: "MESSAGE_PERSISTED"; message?: Message; timestamp: string }

async function readSSEBody(
  res: Response,
  onEvent: (e: StreamEvent) => void,
): Promise<void> {
  if (!res.body) throw new Error("No response body from stream endpoint")

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ""

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break

      buf += decoder.decode(value, { stream: true })

      let sep: number
      while ((sep = buf.indexOf("\n\n")) !== -1) {
        const frame = buf.slice(0, sep)
        buf = buf.slice(sep + 2)

        const dataLine = frame.split("\n").find((l) => l.startsWith("data: "))
        if (!dataLine) continue

        try {
          const ev = JSON.parse(dataLine.slice(6)) as StreamEvent
          onEvent(ev)
        } catch {
          // Skip malformed JSON — keep the stream going.
        }
      }
    }
  } finally {
    reader.releaseLock()
  }
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException("Aborted", "AbortError"))
      return
    }
    const t = setTimeout(resolve, ms)
    const onAbort = () => {
      clearTimeout(t)
      reject(new DOMException("Aborted", "AbortError"))
    }
    signal?.addEventListener("abort", onAbort, { once: true })
  })
}

function isAbortError(e: unknown): boolean {
  return e instanceof DOMException && e.name === "AbortError"
}

function isTerminalStreamEvent(ev: StreamEvent): "ok" | "error" | null {
  if (ev.type === StreamEventType.RUN_ERROR) return "error"
  if (
    ev.type === StreamEventType.MESSAGE_PERSISTED ||
    ev.type === StreamEventType.RUN_FINISHED
  ) {
    return "ok"
  }
  return null
}

const RESUME_BACKOFF_MS = [400, 800, 1500, 2500, 4000] as const

/**
 * Reattach to an in-progress run, retrying while the API is down or the SSE
 * drops mid-stream. Callers keep the same onEvent so tokens continue in place.
 */
export async function followRunningStream(
  conversationId: string,
  onEvent: (e: StreamEvent) => void,
  signal?: AbortSignal,
): Promise<"completed" | "failed" | "not_running" | "aborted"> {
  let terminal: "ok" | "error" | null = null
  const wrap = (ev: StreamEvent) => {
    const t = isTerminalStreamEvent(ev)
    if (t) terminal = t
    onEvent(ev)
  }

  let attempt = 0
  while (!signal?.aborted) {
    try {
      const outcome = await resumeStream(conversationId, wrap, signal)
      if (terminal === "error") return "failed"
      if (terminal === "ok") return "completed"
      if (outcome === "not_running") return "not_running"
      // empty / resumed without terminal: SSE ended while run may still be live
    } catch (e) {
      if (isAbortError(e) || signal?.aborted) return "aborted"
      // API restart / network blip — retry
    }

    const delay = RESUME_BACKOFF_MS[Math.min(attempt, RESUME_BACKOFF_MS.length - 1)]
    attempt++
    try {
      await sleep(delay, signal)
    } catch {
      return "aborted"
    }
  }
  return "aborted"
}

/**
 * POST /api/conversations/{id}/messages
 *
 * Sends the user message and streams SSE. If the connection drops before a
 * terminal event (e.g. API restart), automatically follows via /resume until
 * the run finishes — transparent to the user on the same chat view.
 */
export async function streamMessage(
  conversationId: string,
  content: string,
  onEvent: (e: StreamEvent) => void,
  signal?: AbortSignal,
): Promise<"completed" | "failed" | "not_running" | "aborted"> {
  let terminal: "ok" | "error" | null = null
  const wrap = (ev: StreamEvent) => {
    const t = isTerminalStreamEvent(ev)
    if (t) terminal = t
    onEvent(ev)
  }

  let streamStarted = false
  try {
    const base = await getApiBase()
    const res = await fetch(`${base}/conversations/${conversationId}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content }),
      signal,
    })

    if (!res.ok) {
      const text = await res.text().catch(() => "")
      throw new Error(`Stream failed: ${res.status}${text ? ` — ${text}` : ""}`)
    }
    streamStarted = true
    await readSSEBody(res, wrap)
  } catch (e) {
    if (isAbortError(e) || signal?.aborted) return "aborted"
    if (terminal === "error") return "failed"
    if (terminal === "ok") return "completed"
    if (!streamStarted) throw e
    // Connection lost after the run started — reconnect via resume.
    return followRunningStream(conversationId, onEvent, signal)
  }

  if (terminal === "error") return "failed"
  if (terminal === "ok") return "completed"
  return followRunningStream(conversationId, onEvent, signal)
}

/**
 * POST /api/conversations/{id}/resume
 *
 * Reattaches to an in-progress agent stream (server uses stored stream id + offset).
 * Resolves without calling onEvent when the conversation is not running (409) or
 * the stream ended before subscribe (204).
 */
export async function resumeStream(
  conversationId: string,
  onEvent: (e: StreamEvent) => void,
  signal?: AbortSignal,
): Promise<"resumed" | "not_running" | "empty"> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}/resume`, {
    method: "POST",
    signal,
  })

  if (res.status === 409) return "not_running"
  if (res.status === 204) return "empty"
  if (!res.ok) {
    const text = await res.text().catch(() => "")
    throw new Error(`Resume failed: ${res.status}${text ? ` — ${text}` : ""}`)
  }
  await readSSEBody(res, onEvent)
  return "resumed"
}
