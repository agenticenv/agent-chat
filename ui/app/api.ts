const DEFAULT_API_BASE = "/api"

let configPromise: Promise<string> | null = null

async function getApiBase(): Promise<string> {
  if (configPromise) return configPromise
  configPromise = (async () => {
    try {
      const res = await fetch("/config.json", { cache: "no-store" })
      if (res.ok) {
        const c = (await res.json()) as { apiBase?: string }
        if (c.apiBase) return c.apiBase
      }
    } catch {
      /* ignore */
    }
    return DEFAULT_API_BASE
  })()
  return configPromise
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
