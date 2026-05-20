import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { MessageDTO, UserConvDTO } from '@/api/types';
import type { Notice } from '@/types';
import { MSG_TYPE_SYSTEM } from '../models/messageModel';
import { handleRealtimeMessage, type RealtimeHandlerContext } from '../models/realtimeModel';

function message(id: number, overrides: Partial<MessageDTO> = {}): MessageDTO {
  return {
    id,
    conversation_id: 1,
    seq: id,
    sender_id: 2,
    msg_type: 1,
    content: `message-${id}`,
    reply_to: 0,
    client_id: '',
    created_at: 100 + id,
    ...overrides,
  };
}

function conversation(id: number, overrides: Partial<UserConvDTO> = {}): UserConvDTO {
  return {
    conversation_id: id,
    is_pinned: false,
    is_muted: false,
    unread_count: 0,
    last_msg_at: 1,
    ...overrides,
  };
}

function applySetter<T>(current: T, update: T | ((prev: T) => T)): T {
  return typeof update === 'function' ? (update as (prev: T) => T)(current) : update;
}

function createContext(overrides: Partial<RealtimeHandlerContext> = {}) {
  const state = {
    notice: null as Notice,
    messages: {} as Record<number, MessageDTO[]>,
    lastMsgMap: {} as Record<number, string>,
    conversations: [conversation(1, { unread_count: 2 })],
    typing: {} as Record<number, string>,
    onlineMap: {} as Record<number, boolean>,
  };
  const ctx: RealtimeHandlerContext = {
    currentUID: 1,
    ws: { requestOnlineFriends: vi.fn() } as unknown as RealtimeHandlerContext['ws'],
    typingTimers: { current: {} },
    deletedMessageIDs: { current: new Set() },
    getMessages: () => state.messages,
    getDisplayName: (uid) => `用户${uid}`,
    applyIncomingMessage: vi.fn((msg: MessageDTO) => {
      state.messages[msg.conversation_id] = [...(state.messages[msg.conversation_id] ?? []), msg];
    }),
    setLastMessagePreview: vi.fn((conversationID, msg) => {
      state.lastMsgMap[conversationID] = msg?.content ?? '';
    }),
    setNotice: vi.fn((update) => { state.notice = applySetter(state.notice, update); }),
    setMessages: vi.fn((update) => { state.messages = applySetter(state.messages, update); }),
    setLastMsgMap: vi.fn((update) => { state.lastMsgMap = applySetter(state.lastMsgMap, update); }),
    setConversations: vi.fn((update) => { state.conversations = applySetter(state.conversations, update); }),
    setTyping: vi.fn((update) => { state.typing = applySetter(state.typing, update); }),
    setOnlineMap: vi.fn((update) => { state.onlineMap = applySetter(state.onlineMap, update); }),
    setMention: vi.fn(),
    refreshBase: vi.fn(async () => undefined),
    handleCallWs: vi.fn(),
    ...overrides,
  };
  return { ctx, state };
}

describe('handleRealtimeMessage', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('连接成功后请求好友在线状态并通知callStore', () => {
    const { ctx } = createContext();

    handleRealtimeMessage({ type: 'system', action: 'connected' }, ctx);

    expect(ctx.ws.requestOnlineFriends).toHaveBeenCalledTimes(1);
    expect(ctx.handleCallWs).toHaveBeenCalledWith({ type: 'system', action: 'connected' });
  });

  it('错误消息展示服务端错误文案', () => {
    const { ctx, state } = createContext();

    handleRealtimeMessage({ type: 'error', error: { code: 'bad', message: '失败了' } }, ctx);

    expect(state.notice).toEqual({ kind: 'error', text: '失败了' });
  });

  it('新消息和发送 ack 都归一化后进入统一入站消息流程', () => {
    const { ctx } = createContext();

    handleRealtimeMessage({
      type: 'message',
      action: 'new',
      data: { message_id: 10, conversation_id: 1, sender_id: 2, content: '你好', created_at: 123 },
    }, ctx);
    handleRealtimeMessage({
      type: 'ack',
      action: 'send',
      data: { id: 11, conversation_id: 1, sender_id: 1, content: '收到', created_at: 124 },
    }, ctx);

    expect(ctx.applyIncomingMessage).toHaveBeenCalledTimes(2);
    expect(ctx.applyIncomingMessage).toHaveBeenNthCalledWith(1, expect.objectContaining({ id: 10, content: '你好' }));
    expect(ctx.applyIncomingMessage).toHaveBeenNthCalledWith(2, expect.objectContaining({ id: 11, content: '收到' }));
  });

  it('撤回最新他人消息时标记消息、回退预览并减少未读', () => {
    const { ctx, state } = createContext();
    state.messages = { 1: [message(1), message(2, { content: '最后一条' })] };

    handleRealtimeMessage({
      type: 'message',
      action: 'revoked',
      data: { conversation_id: 1, message_id: 2, sender_id: 2, is_latest: true },
    }, ctx);

    expect(state.messages[1][1].revoked).toBe(true);
    expect(state.lastMsgMap[1]).toBe('消息已撤回');
    expect(state.conversations[0].unread_count).toBe(1);
  });

  it('输入中事件忽略自己，并在超时后清空他人输入态', () => {
    vi.useFakeTimers();
    const { ctx, state } = createContext();

    handleRealtimeMessage({ type: 'typing', action: 'indicator', data: { conversation_id: 1, user_id: 1 } }, ctx);
    expect(state.typing[1]).toBeUndefined();

    handleRealtimeMessage({ type: 'typing', action: 'indicator', data: { conversation_id: 1, user_id: 2 } }, ctx);
    expect(state.typing[1]).toBe('用户2 正在输入...');

    vi.advanceTimersByTime(6000);
    expect(state.typing[1]).toBe('');
  });

  it('在线好友 ack 和 presence 推送合并在线状态', () => {
    const { ctx, state } = createContext();

    handleRealtimeMessage({ type: 'ack', action: 'online_friends', data: { 2: true, 3: false, bad: true } }, ctx);
    handleRealtimeMessage({ type: 'presence', action: 'online', data: { uid: 3 } }, ctx);
    handleRealtimeMessage({ type: 'presence', action: 'offline', data: { uid: 2 } }, ctx);

    expect(state.onlineMap).toEqual({ 2: false, 3: true });
  });

  it('群解散事件追加一次系统消息并保留会话列表', () => {
    vi.spyOn(Date, 'now').mockReturnValue(999);
    const { ctx, state } = createContext();

    handleRealtimeMessage({ type: 'conversation', action: 'group_dissolved', data: { conversation_id: 1, operator_id: 2 } }, ctx);
    handleRealtimeMessage({ type: 'conversation', action: 'group_dissolved', data: { conversation_id: 1, operator_id: 2 } }, ctx);

    expect(ctx.applyIncomingMessage).toHaveBeenCalledTimes(1);
    expect(state.messages[1][0]).toEqual(expect.objectContaining({
      id: 999,
      conversation_id: 1,
      msg_type: MSG_TYPE_SYSTEM,
      content: '群聊已解散',
      client_id: 'system-dissolve-1-999',
    }));
    expect(state.conversations).toEqual([conversation(1, { unread_count: 2 })]);
    expect(state.notice).toEqual({ kind: 'info', text: '群聊已解散' });
  });

  it('好友、成员和普通会话事件展示提示并刷新基础数据', () => {
    const { ctx, state } = createContext();

    handleRealtimeMessage({ type: 'friend', action: 'request' }, ctx);

    expect(state.notice).toEqual({ kind: 'info', text: '收到新的好友申请' });
    expect(ctx.refreshBase).toHaveBeenCalledTimes(1);
  });

  it('将 call 类型消息路由到 handleCallWs', () => {
    const { ctx } = createContext();

    handleRealtimeMessage({ type: 'call', action: 'incoming', data: { call_id: 'c1', caller_uid: 2, call_type: 1, caller_nickname: 'Bob', caller_avatar: '' } }, ctx);

    expect(ctx.handleCallWs).toHaveBeenCalledWith({ type: 'call', action: 'incoming', data: { call_id: 'c1', caller_uid: 2, call_type: 1, caller_nickname: 'Bob', caller_avatar: '' } });
  });

  it('将 call/accepted 路由到 handleCallWs', () => {
    const { ctx } = createContext();

    handleRealtimeMessage({ type: 'call', action: 'accepted', data: { call_id: 'c1' } }, ctx);

    expect(ctx.handleCallWs).toHaveBeenCalledWith({ type: 'call', action: 'accepted', data: { call_id: 'c1' } });
  });

  it('将 call/ended 路由到 handleCallWs', () => {
    const { ctx } = createContext();

    handleRealtimeMessage({ type: 'call', action: 'ended', data: { call_id: 'c1', started_at: 100, duration: 60, end_reason: 'hangup' } }, ctx);

    expect(ctx.handleCallWs).toHaveBeenCalledWith({ type: 'call', action: 'ended', data: { call_id: 'c1', started_at: 100, duration: 60, end_reason: 'hangup' } });
  });
});
