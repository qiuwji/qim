import { getToken } from './http';
import type { MessageDTO, WsResponse } from './types';

export type WsListener = (message: WsResponse) => void;

function wsBaseURL(): string {
  const configured = import.meta.env.VITE_WS_BASE as string | undefined;
  if (configured) return configured.replace(/\/$/, '');
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
}

export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<WsListener>();
  private retryCount = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private stopped = false;
  private onReconnect?: () => void;

  connect(onReconnect?: () => void) {
    this.stopped = false;
    this.onReconnect = onReconnect;
    this.doConnect();
  }

  private doConnect() {
    if (this.stopped) return;
    const token = getToken();
    if (!token || this.socket?.readyState === WebSocket.OPEN || this.socket?.readyState === WebSocket.CONNECTING) return;
    try {
      this.socket = new WebSocket(`${wsBaseURL()}/ws?token=${encodeURIComponent(token)}`);
    } catch {
      this.scheduleRetry();
      return;
    }
    this.socket.onopen = () => {
      this.retryCount = 0;
    };
    this.socket.onmessage = (event) => {
      try {
        this.emit(JSON.parse(event.data) as WsResponse);
      } catch {
        console.warn('invalid ws message', event.data);
      }
    };
    this.socket.onclose = () => {
      this.emit({ type: 'system', action: 'closed' });
      this.scheduleRetry();
    };
    this.socket.onerror = () => {
      this.emit({ type: 'system', action: 'error', error: { code: 'ws_error', message: 'WebSocket 连接异常' } });
    };
  }

  private scheduleRetry() {
    if (this.stopped) return;
    if (this.retryTimer) clearTimeout(this.retryTimer);
    const delay = Math.min(1000 * Math.pow(2, this.retryCount), 30000);
    this.retryCount++;
    this.retryTimer = setTimeout(() => {
      if (!this.stopped) {
        this.doConnect();
        if (this.retryCount === 1 && this.onReconnect) {
          this.onReconnect();
        }
      }
    }, delay);
  }

  close() {
    this.stopped = true;
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.socket?.close();
    this.socket = null;
  }

  on(listener: WsListener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  send(type: string, action: string, data: unknown = {}) {
    if (this.socket?.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket 未连接');
    }
    this.socket.send(JSON.stringify({ type, action, data }));
  }

  sendMessage(input: { conversation_id: number; content: string; msg_type?: number; reply_to?: number }) {
    const clientID = `${Date.now()}-${crypto.randomUUID?.() ?? Math.random().toString(16).slice(2)}`;
    this.send('msg', 'send', {
      conversation_id: input.conversation_id,
      msg_type: input.msg_type ?? 1,
      content: input.content,
      reply_to: input.reply_to ?? 0,
      client_id: clientID
    });
    return clientID;
  }

  typing(conversationID: number) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.send('msg', 'typing', { conversation_id: conversationID });
    }
  }

  revokeMessage(conversation_id: number, message_id: number) {
    this.send('msg', 'revoke', { conversation_id, message_id });
  }

  listMessages(conversation_id: number, before_seq = 0, limit = 30) {
    this.send('msg', 'list', { conversation_id, before_seq, limit });
  }

  searchMessages(conversation_id: number, keyword: string, limit = 20) {
    this.send('msg', 'search', { conversation_id, keyword, limit });
  }

  sendFriendRequest(to_uid: number, message = '') {
    this.send('friend', 'send_request', { to_uid, message });
  }

  handleFriendRequest(req_id: number, accept: boolean) {
    this.send('friend', 'handle_request', { req_id, accept });
  }

  deleteFriend(friend_uid: number) {
    this.send('friend', 'delete', { friend_uid });
  }

  updateFriendRemark(friend_uid: number, remark: string) {
    this.send('friend', 'update_remark', { friend_uid, remark });
  }

  moveFriendGroup(friend_uid: number, group_id: number) {
    this.send('friend', 'move_group', { friend_uid, group_id });
  }

  listFriendGroups() {
    this.send('friend', 'list_groups');
  }

  createFriendGroup(name: string) {
    this.send('friend', 'create_group', { name });
  }

  renameFriendGroup(group_id: number, name: string) {
    this.send('friend', 'rename_group', { group_id, name });
  }

  deleteFriendGroup(group_id: number) {
    this.send('friend', 'delete_group', { group_id });
  }

  sortFriendGroups(groups: Array<{ group_id: number; sort_order: number }>) {
    this.send('friend', 'sort_groups', { groups });
  }

  setMemberRole(conv_id: number, uid: number, role: number) {
    this.send('conv', 'set_role', { conv_id, uid, role });
  }

  transferOwner(conv_id: number, new_owner_id: number) {
    this.send('conv', 'transfer_owner', { conv_id, new_owner_id });
  }

  private emit(message: WsResponse) {
    this.listeners.forEach((listener) => listener(message));
  }
}

export function normalizePushMessage(data: unknown): MessageDTO | null {
  if (!data || typeof data !== 'object') return null;
  const item = data as Record<string, unknown>;
  const conversationID = Number(item.conversation_id ?? item.ConversationID ?? 0);
  const id = Number(item.id ?? item.message_id ?? item.MessageID ?? 0);
  if (!conversationID || !id) return null;
  return {
    id,
    conversation_id: conversationID,
    seq: Number(item.seq ?? item.Seq ?? 0),
    sender_id: Number(item.sender_id ?? item.SenderID ?? 0),
    msg_type: Number(item.msg_type ?? item.MsgType ?? 1),
    content: String(item.content ?? item.Content ?? ''),
    reply_to: Number(item.reply_to ?? item.ReplyTo ?? 0),
    client_id: String(item.client_id ?? item.ClientID ?? ''),
    created_at: Number(item.created_at ?? item.CreatedAt ?? Math.floor(Date.now() / 1000)),
    revoked: Boolean(item.revoked ?? false),
    edited: Boolean(item.edited ?? false)
  };
}
