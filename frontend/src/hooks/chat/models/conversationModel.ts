import type { ConversationDTO, MessageDTO, UserConvDTO } from '@/api/types';

export function sortConversations(list: UserConvDTO[]): UserConvDTO[] {
  return [...list].sort((a, b) => Number(b.is_pinned) - Number(a.is_pinned) || b.last_msg_at - a.last_msg_at);
}

export function groupConversations(list: UserConvDTO[], details: Record<number, ConversationDTO>): UserConvDTO[] {
  return list.filter((item) => details[item.conversation_id]?.type === 2);
}

export function hiddenConversationStorageKey(uid: number): string {
  return `qim:hidden_conversations:${uid}`;
}

export function readHiddenConversationIDs(storageKey: string): number[] {
  try {
    const raw = localStorage.getItem(storageKey);
    const parsed = raw ? JSON.parse(raw) : [];
    if (!Array.isArray(parsed)) return [];
    return parsed.map(Number).filter((id) => Number.isFinite(id) && id > 0);
  } catch {
    return [];
  }
}

export function persistHiddenConversationID(storageKey: string, hiddenIDs: Set<number>, conversationID: number) {
  hiddenIDs.add(conversationID);
  localStorage.setItem(storageKey, JSON.stringify([...hiddenIDs]));
}

export function forgetHiddenConversationID(storageKey: string, hiddenIDs: Set<number>, conversationID: number) {
  if (!hiddenIDs.delete(conversationID)) return;
  localStorage.setItem(storageKey, JSON.stringify([...hiddenIDs]));
}

export function visibleConversations(list: UserConvDTO[], hiddenIDs: Set<number>): UserConvDTO[] {
  return list.filter((item) => !hiddenIDs.has(item.conversation_id));
}

export function markConversationReadLocally(list: UserConvDTO[], cid: number): UserConvDTO[] {
  return list.map((item) => (item.conversation_id === cid ? { ...item, unread_count: 0 } : item));
}

export function decrementConversationUnread(list: UserConvDTO[], cid: number): UserConvDTO[] {
  return list.map((item) => (
    item.conversation_id === cid ? { ...item, unread_count: Math.max(0, item.unread_count - 1) } : item
  ));
}

export function applyIncomingConversation(list: UserConvDTO[], msg: MessageDTO, selectedID: number | null): UserConvDTO[] {
  const unreadCount = selectedID === msg.conversation_id ? 0 : 1;
  const found = list.some((item) => item.conversation_id === msg.conversation_id);
  const mapped = list.map((item) => (
    item.conversation_id === msg.conversation_id
      ? {
          ...item,
          last_msg_at: msg.created_at,
          unread_count: selectedID === msg.conversation_id ? 0 : item.unread_count + 1,
        }
      : item
  ));
  return found
    ? mapped
    : [{ conversation_id: msg.conversation_id, is_pinned: false, is_muted: false, unread_count: unreadCount, last_msg_at: msg.created_at }, ...mapped];
}
