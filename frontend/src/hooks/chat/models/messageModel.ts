import type { MessageDTO, UserConvDTO } from '@/api/types';
import { callTypeLabel, formatDuration } from './callModel';

export const MESSAGE_PAGE_SIZE = 20;
export const REVOKED_MESSAGE_PREVIEW = '消息已撤回';
export const MSG_TYPE_TEXT = 1;
export const MSG_TYPE_IMAGE = 2;
export const MSG_TYPE_LEGACY_SYSTEM = 3;
export const MSG_TYPE_SYSTEM = 5;
export const MSG_TYPE_CALL_RECORD = 6;

export interface CallRecordContent {
  call_id: string;
  call_type: 1 | 2;
  duration: number;
  end_reason: string;
  status: 1 | 2;
}

export function isCallRecord(msg: MessageDTO): boolean {
  return msg.msg_type === MSG_TYPE_CALL_RECORD;
}

export function parseCallRecordContent(content: string): CallRecordContent | null {
  try {
    const parsed = JSON.parse(content);
    if (!parsed || typeof parsed !== 'object') return null;
    const callType = parsed.call_type;
    if (callType !== 1 && callType !== 2) return null;
    return {
      call_id: String(parsed.call_id ?? ''),
      call_type: callType,
      duration: Number(parsed.duration ?? 0),
      end_reason: String(parsed.end_reason ?? ''),
      status: parsed.status === 2 ? 2 : 1,
    };
  } catch {
    return null;
  }
}

export function endReasonText(reason: string): string {
  switch (reason) {
    case 'rejected': return '已拒绝';
    case 'timeout': return '无人接听';
    case 'cancelled': return '已取消';
    case 'disconnect': return '连接断开';
    default: return '';
  }
}

export function callRecordLabel(callType: 1 | 2, _status: 1 | 2): string {
  const label = callTypeLabel(callType);
  return label;
}

export function callRecordSummary(content: string): string {
  const parsed = parseCallRecordContent(content);
  if (!parsed) return '[通话记录]';
  const label = callTypeLabel(parsed.call_type);
  if (parsed.status === 2) {
    return `[${label}] ${formatDuration(parsed.duration)}`;
  }
  const reason = endReasonText(parsed.end_reason);
  return `[${label}]${reason ? ` ${reason}` : ''}`;
}

export function messageDisplayText(msg: MessageDTO): string {
  if (msg.msg_type === MSG_TYPE_IMAGE) return '[图片]';
  if (msg.msg_type === MSG_TYPE_CALL_RECORD) return callRecordSummary(msg.content);
  return msg.content;
}

export function isMentionedMe(msg: MessageDTO, currentUID: number): boolean {
  if (msg.mention_all) return true;
  return !!msg.mention_uids?.includes(currentUID);
}

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
  if (msg.msg_type === MSG_TYPE_IMAGE) return '[图片]';
  if (msg.msg_type === MSG_TYPE_CALL_RECORD) return callRecordSummary(msg.content);
  return msg.content || undefined;
}

export function isSystemMessage(msg: MessageDTO): boolean {
  return msg.msg_type === MSG_TYPE_SYSTEM || msg.msg_type === MSG_TYPE_LEGACY_SYSTEM;
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
