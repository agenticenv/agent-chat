import { useState, useEffect, useCallback, useRef } from "react"
import {
  getConversations,
  createConversation,
  getMessages,
  sendMessage,
  renameConversation,
  deleteConversation,
  type Conversation,
  type Message,
} from "../api"

const PlusIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M5 12h14M12 5v14" strokeLinecap="round" />
  </svg>
)
const MessageIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
  </svg>
)
const SendIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="m22 2-7 20-4-9-9-4Z" />
    <path d="M22 2 11 13" />
  </svg>
)
const MoreIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="1.5" />
    <circle cx="6" cy="12" r="1.5" />
    <circle cx="18" cy="12" r="1.5" />
  </svg>
)
const PencilIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
  </svg>
)
const TrashIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M3 6h18M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
    <line x1="10" y1="11" x2="10" y2="17" />
    <line x1="14" y1="11" x2="14" y2="17" />
  </svg>
)
const SunIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="4" />
    <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
  </svg>
)
const MoonIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
  </svg>
)

function PromptInput({
  value,
  onChange,
  onKeyDown,
  onSend,
}: {
  value: string
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void
  onSend: () => void
}) {
  const ref = useRef<HTMLTextAreaElement>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    el.style.height = "auto"
    const maxH = 8 * 24
    el.style.height = `${Math.min(el.scrollHeight, maxH)}px`
  }, [value])

  return (
    <div className="flex items-end gap-2 rounded-xl border border-border bg-muted/30 px-4 py-2">
      <textarea
        ref={ref}
        placeholder="Type a message..."
        rows={1}
        className="min-h-[44px] max-h-[192px] flex-1 resize-none overflow-y-auto border-0 bg-transparent py-2.5 text-base outline-none placeholder:text-muted-foreground cursor-text"
        value={value}
        onChange={onChange}
        onKeyDown={onKeyDown}
      />
      <button
        onClick={onSend}
        className="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-lg bg-primary text-primary-foreground hover:opacity-90"
        aria-label="Send"
      >
        <SendIcon />
      </button>
    </div>
  )
}

type ThemeMode = "light" | "dark"

function applyTheme(mode: ThemeMode) {
  if (typeof window === "undefined") return
  localStorage.setItem("theme", mode)
  document.documentElement.classList.toggle("dark", mode === "dark")
}

function getEffectiveTheme(): ThemeMode {
  if (typeof window === "undefined" || typeof localStorage === "undefined") return "light"
  const t = localStorage.getItem("theme")
  if (t === "dark" || t === "light") return t
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

export default function AssistantPage() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [input, setInput] = useState("")
  const [loading, setLoading] = useState(true)
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [menuOpenId, setMenuOpenId] = useState<string | null>(null)
  const [renameConv, setRenameConv] = useState<{ id: string; title: string } | null>(null)
  const [renameValue, setRenameValue] = useState("")
  const [deleteConvId, setDeleteConvId] = useState<string | null>(null)
  const [themeMode, setThemeMode] = useState<ThemeMode>("light")

  useEffect(() => {
    setThemeMode(getEffectiveTheme())
  }, [])
  const menuRef = useRef<HTMLDivElement>(null)

  const handleThemeToggle = () => {
    const next = themeMode === "dark" ? "light" : "dark"
    setThemeMode(next)
    applyTheme(next)
  }

  const loadConversations = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await getConversations()
      setConversations(list)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load conversations")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadConversations()
  }, [loadConversations])

  useEffect(() => {
    if (!selectedId) {
      setMessages([])
      return
    }
    let cancelled = false
    setLoadingMessages(true)
    getMessages(selectedId)
      .then((list) => {
        if (!cancelled) setMessages(list)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load messages")
      })
      .finally(() => {
        if (!cancelled) setLoadingMessages(false)
      })
    return () => {
      cancelled = true
    }
  }, [selectedId])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpenId(null)
      }
    }
    if (menuOpenId) {
      document.addEventListener("mousedown", handleClickOutside)
      return () => document.removeEventListener("mousedown", handleClickOutside)
    }
  }, [menuOpenId])

  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => {
      if (typeof localStorage !== "undefined" && !localStorage.getItem("theme")) {
        setThemeMode(mq.matches ? "dark" : "light")
        document.documentElement.classList.toggle("dark", mq.matches)
      }
    }
    mq.addEventListener("change", handler)
    return () => mq.removeEventListener("change", handler)
  }, [])

  const handleNewConversation = () => {
    setError(null)
    setMenuOpenId(null)
    setSelectedId(null)
    setMessages([])
  }

  const handleSelectConversation = (id: string | null) => {
    setMenuOpenId(null)
    setSelectedId(id)
  }

  const handleRenameOpen = (conv: Conversation) => {
    setMenuOpenId(null)
    setRenameConv({ id: conv.id, title: conv.title })
    setRenameValue(conv.title)
  }

  const handleRenameSave = async () => {
    if (!renameConv || !renameValue.trim()) return
    setError(null)
    try {
      await renameConversation(renameConv.id, renameValue.trim())
      setConversations((prev) =>
        prev.map((c) => (c.id === renameConv.id ? { ...c, title: renameValue.trim() } : c))
      )
      setRenameConv(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to rename")
    }
  }

  const handleDeleteOpen = (id: string) => {
    setMenuOpenId(null)
    setDeleteConvId(id)
  }

  const handleDeleteConfirm = async () => {
    if (!deleteConvId) return
    setError(null)
    try {
      await deleteConversation(deleteConvId)
      let nextSelection: string | null | undefined = undefined
      setConversations((prev) => {
        const next = prev.filter((c) => c.id !== deleteConvId)
        if (selectedId === deleteConvId) nextSelection = next[0]?.id ?? null
        return next
      })
      if (nextSelection !== undefined) {
        setSelectedId(nextSelection)
        setMessages([])
      }
      setDeleteConvId(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete")
    }
  }

  const handleSend = async () => {
    const text = input.trim()
    if (!text) return
    if (!selectedId) {
      try {
        const conv = await createConversation()
        setConversations((prev) => [conv, ...prev])
        setSelectedId(conv.id)
        const userMsg: Message = { id: `temp-${Date.now()}`, role: "user", content: text, createdAt: new Date().toISOString() }
        setMessages([userMsg])
        setInput("")
        const msg = await sendMessage(conv.id, text)
        setMessages([userMsg, msg])
        setConversations((prev) =>
          prev.map((c) =>
            c.id === conv.id ? { ...c, title: text.slice(0, 32) + (text.length > 32 ? "…" : "") } : c
          )
        )
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to send")
      }
      return
    }
    const userMsg: Message = { id: `temp-${Date.now()}`, role: "user", content: text, createdAt: new Date().toISOString() }
    setMessages((prev) => [...prev, userMsg])
    setInput("")
    try {
      const msg = await sendMessage(selectedId, text)
      setMessages((prev) => [...prev, msg])
      const conv = conversations.find((c) => c.id === selectedId)
      if (conv?.title === "New conversation") {
        setConversations((prev) =>
          prev.map((c) =>
            c.id === selectedId ? { ...c, title: text.slice(0, 32) + (text.length > 32 ? "…" : "") } : c
          )
        )
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to send message")
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const showCenterPrompt = !selectedId || messages.length === 0

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="flex w-64 shrink-0 flex-col border-r border-border bg-muted/30">
        <div className="flex items-center justify-between border-b border-border p-3">
          <h2 className="text-sm font-semibold">Agent demo</h2>
          <button
            type="button"
            onClick={handleThemeToggle}
            className="flex h-8 w-8 cursor-pointer items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
            aria-label={themeMode === "dark" ? "Switch to light mode" : "Switch to dark mode"}
          >
            {themeMode === "dark" ? <MoonIcon /> : <SunIcon />}
          </button>
        </div>
        <div className="p-2">
          <button
            onClick={handleNewConversation}
            className="flex w-full cursor-pointer items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm hover:bg-muted"
          >
            <PlusIcon />
            New conversation
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-auto p-2">
          {loading ? (
            <div className="space-y-2 px-2 py-4">
              <div className="h-3 animate-pulse rounded bg-muted" />
              <div className="h-3 w-4/5 animate-pulse rounded bg-muted" />
              <div className="h-3 w-3/4 animate-pulse rounded bg-muted" />
            </div>
          ) : (
            <div className="space-y-1">
              {conversations.map((c) => (
                <div
                  key={c.id}
                  className={`group flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                    c.id === selectedId
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted"
                  }`}
                >
                  <button
                    onClick={() => handleSelectConversation(c.id)}
                    className="flex min-w-0 flex-1 cursor-pointer items-center gap-3"
                  >
                    <MessageIcon />
                    <span className="truncate">{c.title}</span>
                  </button>
                  <div ref={menuOpenId === c.id ? menuRef : undefined} className="relative shrink-0">
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation()
                        setMenuOpenId((prev) => (prev === c.id ? null : c.id))
                      }}
                      className={`cursor-pointer rounded p-1 opacity-0 group-hover:opacity-100 ${menuOpenId === c.id ? "opacity-100" : ""}`}
                      aria-label="Options"
                    >
                      <MoreIcon />
                    </button>
                    {menuOpenId === c.id && (
                      <div className="absolute right-0 top-full z-50 mt-1 min-w-[130px] rounded-lg border border-border bg-popover py-1 text-popover-foreground shadow-lg">
                        <button
                          type="button"
                          onClick={() => handleRenameOpen(c)}
                          className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground"
                        >
                          <PencilIcon />
                          Rename
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDeleteOpen(c.id)}
                          className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm text-destructive hover:bg-destructive/10 hover:text-destructive"
                        >
                          <TrashIcon />
                          Delete
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </aside>

      {/* Main */}
      <main className="flex flex-1 flex-col min-w-0">
        {error && (
          <div className="bg-destructive/10 px-4 py-2 text-sm text-destructive">{error}</div>
        )}
        {showCenterPrompt ? (
          <div className="flex flex-1 flex-col items-center justify-center px-4">
            <h1 className="mb-8 text-2xl font-semibold">How can I help you today?</h1>
            <div className="mx-auto w-full max-w-2xl">
              <PromptInput
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                onSend={handleSend}
              />
            </div>
          </div>
        ) : (
          <>
            <div className="min-h-0 flex-1 overflow-y-auto">
              <div className="mx-auto max-w-3xl px-4 py-8">
                {loadingMessages ? (
                  <div className="flex flex-col items-center gap-4 py-12">
                    <div className="flex gap-1.5">
                      <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:0ms]" />
                      <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:150ms]" />
                      <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:300ms]" />
                    </div>
                    <p className="text-sm text-muted-foreground">Loading messages…</p>
                  </div>
                ) : (
                  <div className="space-y-8">
                    {messages.map((msg) => (
                      <div
                        key={msg.id}
                        className={`flex gap-4 ${
                          msg.role === "user" ? "justify-end" : "justify-start"
                        }`}
                      >
                        {msg.role === "assistant" && (
                          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted">
                            <MessageIcon />
                          </div>
                        )}
                        <div
                          className={`max-w-[85%] rounded-2xl px-4 py-3 text-[15px] leading-relaxed ${
                            msg.role === "user"
                              ? "bg-primary text-primary-foreground"
                              : "rounded-bl-md bg-muted"
                          }`}
                        >
                          {msg.content}
                        </div>
                        {msg.role === "user" && <div className="w-8 shrink-0" />}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
            <div className="shrink-0 bg-background px-4 py-4">
              <div className="mx-auto max-w-2xl">
                <PromptInput
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  onSend={handleSend}
                />
              </div>
            </div>
          </>
        )}
      </main>

      {/* Rename modal */}
      {renameConv && (
        <div
          className="fixed inset-0 z-50 flex cursor-pointer items-center justify-center bg-black/50"
          onClick={() => setRenameConv(null)}
        >
          <div
            className="w-full max-w-md cursor-default rounded-xl border border-border bg-background p-6 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="mb-4 text-lg font-semibold">Rename conversation</h3>
            <input
              type="text"
              className="mb-6 w-full cursor-text rounded-lg border border-border bg-background px-3 py-2 outline-ring focus:ring-2"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              placeholder="Conversation name"
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRenameSave()
                if (e.key === "Escape") setRenameConv(null)
              }}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setRenameConv(null)}
                className="cursor-pointer rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleRenameSave}
                disabled={!renameValue.trim()}
                className="cursor-pointer rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {deleteConvId && (
        <div
          className="fixed inset-0 z-50 flex cursor-pointer items-center justify-center bg-black/50"
          onClick={() => setDeleteConvId(null)}
        >
          <div
            className="w-full max-w-md cursor-default rounded-xl border border-border bg-background p-6 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="mb-2 text-lg font-semibold text-destructive">
              Delete conversation?
            </h3>
            <p className="mb-6 text-sm text-muted-foreground">
              This action cannot be undone. All messages in this conversation will be permanently
              deleted.
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setDeleteConvId(null)}
                className="cursor-pointer rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteConfirm}
                className="cursor-pointer rounded-lg bg-destructive px-4 py-2 text-sm text-destructive-foreground hover:opacity-90"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
