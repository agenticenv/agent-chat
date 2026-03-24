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

/* ----- Real API ----- */

async function getConversationsReal(): Promise<Conversation[]> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations`)
  if (!res.ok) throw new Error(`Failed to fetch conversations: ${res.status}`)
  const data = (await parseJson(res)) as Conversation[] | { conversations?: Conversation[] }
  return Array.isArray(data) ? data : data.conversations ?? []
}

async function createConversationReal(title = "New conversation"): Promise<Conversation> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error(`Failed to create conversation: ${res.status}`)
  const data = (await parseJson(res)) as { conversation?: Conversation } & Conversation
  const conv = data.conversation ?? data
  if (!conv?.id || !conv?.title) throw new Error("Invalid conversation response")
  return conv
}

async function getMessagesReal(conversationId: string): Promise<Message[]> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}/messages`)
  if (!res.ok) throw new Error(`Failed to fetch messages: ${res.status}`)
  const data = (await parseJson(res)) as Message[] | { messages?: Message[] }
  return Array.isArray(data) ? data : data.messages ?? []
}

async function sendMessageReal(conversationId: string, content: string): Promise<Message> {
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

async function renameConversationReal(
  conversationId: string,
  title: string
): Promise<void> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error(`Failed to rename: ${res.status}`)
}

async function deleteConversationReal(conversationId: string): Promise<void> {
  const base = await getApiBase()
  const res = await fetch(`${base}/conversations/${conversationId}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(`Failed to delete: ${res.status}`)
}

/* ----- Mock API (in-memory when backend unavailable) ----- */

const mockStore: {
  conversations: Conversation[]
  messagesByConv: Map<string, Message[]>
} = { conversations: [], messagesByConv: new Map() }

function mockId() {
  return crypto.randomUUID?.() ?? `mock-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

async function getConversationsMock(): Promise<Conversation[]> {
  return [...mockStore.conversations]
}

async function createConversationMock(title = "New conversation"): Promise<Conversation> {
  const conv: Conversation = { id: mockId(), title }
  mockStore.conversations.unshift(conv)
  mockStore.messagesByConv.set(conv.id, [])
  return conv
}

async function getMessagesMock(conversationId: string): Promise<Message[]> {
  return [...(mockStore.messagesByConv.get(conversationId) ?? [])]
}

async function sendMessageMock(conversationId: string, content: string): Promise<Message> {
  const userMsg: Message = { id: mockId(), role: "user", content }
  const convMsgs = mockStore.messagesByConv.get(conversationId) ?? []
  convMsgs.push(userMsg)
  mockStore.messagesByConv.set(conversationId, convMsgs)

  const conv = mockStore.conversations.find((c) => c.id === conversationId)
  if (conv && conv.title === "New conversation") {
    conv.title = content.slice(0, 32) + (content.length > 32 ? "…" : "")
  }

  const assistantMsg: Message = {
    id: mockId(),
    role: "assistant",
    content: "Backend not connected. Add REST API routes to enable real responses.",
  }
  convMsgs.push(assistantMsg)
  return assistantMsg
}

async function renameConversationMock(conversationId: string, title: string): Promise<void> {
  const conv = mockStore.conversations.find((c) => c.id === conversationId)
  if (conv) conv.title = title
}

async function deleteConversationMock(conversationId: string): Promise<void> {
  mockStore.conversations = mockStore.conversations.filter((c) => c.id !== conversationId)
  mockStore.messagesByConv.delete(conversationId)
}

/* ----- Public API (tries real, falls back to mock on failure) ----- */

let useMock = false

export async function getConversations(): Promise<Conversation[]> {
  if (useMock) return getConversationsMock()
  try {
    return await getConversationsReal()
  } catch {
    useMock = true
    return getConversationsMock()
  }
}

export async function createConversation(title = "New conversation"): Promise<Conversation> {
  if (useMock) return createConversationMock(title)
  try {
    return await createConversationReal(title)
  } catch {
    useMock = true
    return createConversationMock(title)
  }
}

export async function getMessages(conversationId: string): Promise<Message[]> {
  if (useMock) return getMessagesMock(conversationId)
  try {
    return await getMessagesReal(conversationId)
  } catch {
    useMock = true
    return getMessagesMock(conversationId)
  }
}

export async function sendMessage(conversationId: string, content: string): Promise<Message> {
  if (useMock) return sendMessageMock(conversationId, content)
  try {
    return await sendMessageReal(conversationId, content)
  } catch {
    useMock = true
    return sendMessageMock(conversationId, content)
  }
}

export async function renameConversation(
  conversationId: string,
  title: string
): Promise<void> {
  if (useMock) return renameConversationMock(conversationId, title)
  try {
    return await renameConversationReal(conversationId, title)
  } catch {
    useMock = true
    return renameConversationMock(conversationId, title)
  }
}

export async function deleteConversation(conversationId: string): Promise<void> {
  if (useMock) return deleteConversationMock(conversationId)
  try {
    return await deleteConversationReal(conversationId)
  } catch {
    useMock = true
    return deleteConversationMock(conversationId)
  }
}
