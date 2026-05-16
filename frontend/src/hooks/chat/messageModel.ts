import type { MessageDTO, UserConvDTO } from '../../api/types';

export const MESSAGE_PAGE_SIZE = 20;
export const REVOKED_MESSAGE_PREVIEW = '消息已撤回';

export type PreviewUpdate =
  | { kind: 'none' }
  | { kind: 'text'; text: string }
  | { kind: 'message'; message?: MessageDTO };

export function deletedMessageStorageKey(uid: number): string {
  return `qim:deleted_messages:${uid}`;
}

export function readDeletedMessageIDs(storageKey: string): number[] {
  try {
    const raw = localStorage.getItem(storageKey);
    const parsed = raw ? JSON.parse(raw) : [];
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((item) => {
        if (typeof item === 'number') return item;
        if (typeof item === 'string') {
          const parts = item.split(':');
          return parts.length === 2 ? Number(parts[1]) : Number.NaN;
        }
        return Number.NaN;
      })
      .filter((id) => Number.isFinite(id) && id > 0);
  } catch {
    return [];
  }
}

export function persistDeletedMessageID(storageKey: string, deletedIDs: Set<number>, messageID: number) {
  deletedIDs.add(messageID);
  localStorage.setItem(storageKey, JSON.stringify([...deletedIDs]));
}

export function messagePreviewText(msg: MessageDTO | undefined): string | undefined {
  if (!msg) return undefined;
  if (msg.revoked) return REVOKED_MESSAGE_PREVIEW;
  return msg.content || undefined;
}

export function isMessageLocallyDeleted(msg: MessageDTO, deletedIDs: Set<number>): boolean {
  return deletedIDs.has(msg.id);
}

export function visibleMessages(list: MessageDTO[], deletedIDs: Set<number>): MessageDTO[] {
  return list.filter((msg) => !isMessageLocallyDeleted(msg, deletedIDs));
}

export function latestVisibleMessage(list: MessageDTO[], deletedIDs: Set<number>): MessageDTO | undefined {
  return visibleMessages(list, deletedIDs)[0];
}

export function removeMessageByID(list: MessageDTO[], messageID: number): MessageDTO[] {
  return list.filter((msg) => msg.id !== messageID);
}

export function applyMessagePreview(prev: Record<number, string>, cid: number, msg: MessageDTO | undefined): Record<number, string> {
  const next = { ...prev };
  const text = messagePreviewText(msg);
  if (text) next[cid] = text;
  else delete next[cid];
  return next;
}

export function applyPreviewTexts(
  prev: Record<number, string>,
  conversations: UserConvDTO[],
  previews: Record<number, string | undefined>,
): Record<number, string> {
  const next = { ...prev };
  for (const chat of conversations) {
    const text = previews[chat.conversation_id];
    if (text) next[chat.conversation_id] = text;
    else delete next[chat.conversation_id];
  }
  return next;
}

export function mergeLoadedMessages(list: MessageDTO[], existing: MessageDTO[], beforeSeq: number): MessageDTO[] {
  const sorted = [...list].sort((a, b) => a.seq - b.seq);
  return beforeSeq > 0 ? [...sorted, ...existing] : sorted;
}

export function revokedPreviewUpdate(
  currentList: MessageDTO[],
  messageID: number,
  isLatest: boolean,
  deletedIDs: Set<number>,
): PreviewUpdate {
  const lastMsg = currentList[currentList.length - 1];
  if (!isLatest && lastMsg?.id !== messageID) return { kind: 'none' };
  if (deletedIDs.has(messageID)) {
    const nextList = removeMessageByID(currentList, messageID);
    return { kind: 'message', message: nextList[nextList.length - 1] };
  }
  return { kind: 'text', text: REVOKED_MESSAGE_PREVIEW };
}
