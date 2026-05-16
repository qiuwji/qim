import type { MessageDTO, UserConvDTO, WsResponse } from '../../api/types';
import type { RealtimeClient } from '../../api/ws';
import { normalizePushMessage } from '../../api/ws';
import type { Notice } from '../../types';
import { pushText } from '../../utils';
import { decrementConversationUnread } from './conversationModel';
import { revokedPreviewUpdate } from './messageModel';

type StateSetter<T> = (updater: T | ((prev: T) => T)) => void;
type CurrentRef<T> = { current: T };

export interface RealtimeHandlerContext {
  currentUID: number;
  ws: RealtimeClient;
  typingTimers: CurrentRef<Record<number, ReturnType<typeof setTimeout>>>;
  deletedMessageIDs: CurrentRef<Set<number>>;
  getMessages: () => Record<number, MessageDTO[]>;
  getDisplayName: (uid: number) => string;
  applyIncomingMessage: (message: MessageDTO) => void;
  setLastMessagePreview: (conversationID: number, message: MessageDTO | undefined) => void;
  setNotice: StateSetter<Notice>;
  setMessages: StateSetter<Record<number, MessageDTO[]>>;
  setLastMsgMap: StateSetter<Record<number, string>>;
  setConversations: StateSetter<UserConvDTO[]>;
  setTyping: StateSetter<Record<number, string>>;
  setOnlineMap: StateSetter<Record<number, boolean>>;
  refreshBase: () => Promise<void>;
}

export function handleRealtimeMessage(msg: WsResponse, ctx: RealtimeHandlerContext) {
  if (msg.type === 'system') {
    handleSystemMessage(msg, ctx);
    return;
  }
  if (msg.type === 'error') {
    ctx.setNotice({ kind: 'error', text: msg.error?.message || '请求失败' });
    return;
  }
  if ((msg.type === 'ack' && msg.action === 'send') || (msg.type === 'message' && msg.action === 'new')) {
    const next = normalizePushMessage(msg.data);
    if (next) ctx.applyIncomingMessage(next);
    return;
  }
  if (msg.type === 'message' && msg.action === 'revoked') {
    handleRevokedMessage(msg, ctx);
    return;
  }
  if (msg.type === 'typing' && msg.action === 'indicator') {
    handleTypingIndicator(msg, ctx);
    return;
  }
  if (msg.type === 'ack' && msg.action === 'online_friends') {
    handleOnlineFriendsAck(msg, ctx);
    return;
  }
  if (msg.type === 'presence') {
    handlePresencePush(msg, ctx);
    return;
  }
  if (['friend', 'member', 'conversation'].includes(msg.type)) {
    ctx.setNotice({ kind: 'info', text: pushText(msg) });
    void ctx.refreshBase();
  }
}

function handleSystemMessage(msg: WsResponse, ctx: RealtimeHandlerContext) {
  if (msg.action === 'connected') {
    ctx.ws.requestOnlineFriends();
  }
}

function handleRevokedMessage(msg: WsResponse, ctx: RealtimeHandlerContext) {
  const data = msg.data as Record<string, unknown> | undefined;
  const conversationID = Number(data?.conversation_id ?? 0);
  const messageID = Number(data?.message_id ?? 0);
  const senderID = Number(data?.sender_id ?? 0);
  const isLatest = Boolean(data?.is_latest);
  const currentList = ctx.getMessages()[conversationID] ?? [];

  ctx.setMessages((prev) => ({
    ...prev,
    [conversationID]: (prev[conversationID] ?? []).map((item) => (
      item.id === messageID ? { ...item, revoked: true } : item
    )),
  }));

  const previewUpdate = revokedPreviewUpdate(currentList, messageID, isLatest, ctx.deletedMessageIDs.current);
  if (previewUpdate.kind === 'message') {
    ctx.setLastMessagePreview(conversationID, previewUpdate.message);
  } else if (previewUpdate.kind === 'text') {
    ctx.setLastMsgMap((prev) => ({ ...prev, [conversationID]: previewUpdate.text }));
  }
  if (isLatest && senderID !== ctx.currentUID) {
    ctx.setConversations((prev) => decrementConversationUnread(prev, conversationID));
  }
}

function handleTypingIndicator(msg: WsResponse, ctx: RealtimeHandlerContext) {
  const data = msg.data as Record<string, unknown> | undefined;
  const conversationID = Number(data?.conversation_id ?? 0);
  const uid = Number(data?.user_id ?? 0);
  if (!conversationID || uid === ctx.currentUID) return;

  ctx.setTyping((prev) => ({ ...prev, [conversationID]: `${ctx.getDisplayName(uid)} 正在输入...` }));
  if (ctx.typingTimers.current[conversationID]) {
    clearTimeout(ctx.typingTimers.current[conversationID]);
  }
  ctx.typingTimers.current[conversationID] = setTimeout(() => {
    ctx.setTyping((prev) => ({ ...prev, [conversationID]: '' }));
  }, 6000);
}

function handleOnlineFriendsAck(msg: WsResponse, ctx: RealtimeHandlerContext) {
  const data = msg.data as Record<string, unknown> | undefined;
  if (!data) return;

  const next: Record<number, boolean> = {};
  for (const [key, value] of Object.entries(data)) {
    const uid = Number(key);
    if (uid && typeof value === 'boolean') next[uid] = value;
  }
  ctx.setOnlineMap((prev) => ({ ...prev, ...next }));
}

function handlePresencePush(msg: WsResponse, ctx: RealtimeHandlerContext) {
  const data = msg.data as Record<string, unknown> | undefined;
  const uid = Number(data?.uid ?? 0);
  if (!uid) return;

  ctx.setOnlineMap((prev) => ({ ...prev, [uid]: msg.action === 'online' }));
}
