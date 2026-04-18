const DEFAULT_API_BASE = "/api"

export interface ClientRuntimeConfig {
  apiBase: string
  /** When false, use REST POST /messages instead of SSE /messages/stream. */
  enableStream: boolean
}

let runtimeConfigPromise: Promise<ClientRuntimeConfig> | null = null

/**
 * Browser runtime config: from `config.json` in production (Docker entrypoint),
 * or from `VITE_ENABLE_STREAM` in dev (no config.json).
 */
export async function getRuntimeConfig(): Promise<ClientRuntimeConfig> {
  if (runtimeConfigPromise) return runtimeConfigPromise
  runtimeConfigPromise = (async () => {
    if (import.meta.env.DEV) {
      return {
        apiBase: DEFAULT_API_BASE,
        enableStream: import.meta.env.VITE_ENABLE_STREAM !== "false",
      }
    }
    try {
      const res = await fetch("/config.json", { cache: "no-store" })
      if (res.ok) {
        const c = (await res.json()) as { apiBase?: string; enableStream?: boolean }
        return {
          apiBase: typeof c.apiBase === "string" && c.apiBase ? c.apiBase : DEFAULT_API_BASE,
          enableStream: typeof c.enableStream === "boolean" ? c.enableStream : true,
        }
      }
    } catch {
      /* ignore */
    }
    return { apiBase: DEFAULT_API_BASE, enableStream: true }
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

export interface Conversation {
  id: string
  title: string
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

export async function sendMessage(conversationId: string, content: string): Promise<Message> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}/messages`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  })
  if (!res.ok) throw new Error(`Failed to send message: ${res.status}`)
  const data = (await parseJson(res)) as { message?: Message } & Message
  const msg = data.message ?? data
  if (!msg?.id || !msg?.role || !msg?.content) throw new Error("Invalid message response")
  return msg
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

// ── Streaming ─────────────────────────────────────────────────────────────────

export type StreamEvent =
  | { type: "token"; content: string; timestamp: string }
  | { type: "tool_call"; tool_name: string; tool_call_id?: string; timestamp: string }
  | { type: "tool_result"; tool_name: string; result: unknown; timestamp: string }
  | { type: "error"; content: string; timestamp: string }
  | { type: "done"; message?: Message; timestamp: string }

/**
 * POST /api/conversations/{id}/messages/stream
 *
 * Sends the user message and calls onEvent for each SSE frame the server
 * emits. Uses fetch() + ReadableStream (not EventSource) because we need to
 * POST a request body.
 *
 * The server runs the agent in a background goroutine independent of this
 * HTTP connection, so aborting via signal does NOT cancel the agent — it only
 * stops receiving events. The final state is always retrievable via getMessages().
 */
export async function streamMessage(
  conversationId: string,
  content: string,
  onEvent: (e: StreamEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}/messages/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
    signal,
  })

  if (!res.ok) {
    const text = await res.text().catch(() => "")
    throw new Error(`Stream failed: ${res.status}${text ? ` — ${text}` : ""}`)
  }
  if (!res.body) throw new Error("No response body from stream endpoint")

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ""

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break

      buf += decoder.decode(value, { stream: true })

      // SSE frames are separated by a blank line (\n\n).
      // A single read() may contain multiple frames or a partial frame.
      let sep: number
      while ((sep = buf.indexOf("\n\n")) !== -1) {
        const frame = buf.slice(0, sep)
        buf = buf.slice(sep + 2)

        // Find the data line inside the frame (SSE allows multi-line frames
        // with "data: " prefix; we only emit single-line data frames).
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
