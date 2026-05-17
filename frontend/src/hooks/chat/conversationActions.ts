import { api } from '@/api/http';
import type { ChatStoreDeps } from './types';

export function createConversationActions(d: ChatStoreDeps) {
  async function togglePin(convID?: number) {
    const id = convID ?? d.selectedID;
    if (!id) return;
    let nextPinned = false;
    d.setConversations((prev) => {
      const conv = prev.find((c) => c.conversation_id === id);
      if (!conv) return prev;
      nextPinned = !conv.is_pinned;
      return prev.map((i) => (i.conversation_id === id ? { ...i, is_pinned: nextPinned } : i));
    });
    try { await api.pinChat(id, nextPinned); }
    catch (err) { d.setConversations((p) => p.map((i) => (i.conversation_id === id ? { ...i, is_pinned: !nextPinned } : i))); d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '置顶失败' }); }
  }

  async function toggleMute(convID?: number) {
    const id = convID ?? d.selectedID;
    if (!id) return;
    let nextMuted = false;
    d.setConversations((prev) => {
      const conv = prev.find((c) => c.conversation_id === id);
      if (!conv) return prev;
      nextMuted = !conv.is_muted;
      return prev.map((i) => (i.conversation_id === id ? { ...i, is_muted: nextMuted } : i));
    });
    try { await api.muteChat(id, nextMuted); }
    catch (err) { d.setConversations((p) => p.map((i) => (i.conversation_id === id ? { ...i, is_muted: !nextMuted } : i))); d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '免打扰设置失败' }); }
  }

  async function markAllRead() {
    try { await api.markAllRead(); d.setConversations((p) => p.map((i) => ({ ...i, unread_count: 0 }))); }
    catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '全部已读失败' }); }
  }

  function selectChat(id: number) {
    if (d.selectedID === id) { d.setSelectedID(null); d.selectedIDRef.current = null; d.setDetailOpen(false); d.setMobilePane('list'); }
    else { d.openConversation(id); }
    d.setReplyTo(null); d.setChatSearchResult([]);
  }

  return { togglePin, toggleMute, markAllRead, selectChat };
}
