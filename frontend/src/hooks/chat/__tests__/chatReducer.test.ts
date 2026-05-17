import { describe, expect, it } from 'vitest';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatReducer, createInitialChatState } from '../reducers/chatReducer';

const me: UserDTO = { id: 1, username: 'me', nickname: 'Me', avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 };

function conv(id: number, overrides: Partial<UserConvDTO> = {}): UserConvDTO {
  return { conversation_id: id, is_pinned: false, is_muted: false, unread_count: 0, last_msg_at: id, ...overrides };
}

function detail(id: number, type: 1 | 2): ConversationDTO {
  return { id, type, name: `conv-${id}`, avatar: '', owner_id: 1, member_count: 2, member_limit: 500, max_seq: 0, created_at: 1 };
}

function message(id: number, conversationID = 1, overrides: Partial<MessageDTO> = {}): MessageDTO {
  return {
    id,
    conversation_id: conversationID,
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

const friend: FriendDTO = { id: 1, friend_uid: 2, remark: 'Friend', group_id: 0, created_at: 1 };
const pendingIncoming: FriendRequestDTO = { id: 1, from_uid: 2, to_uid: 1, message: '', status: 0, created_at: 1 };
const doneOutgoing: FriendRequestDTO = { id: 2, from_uid: 1, to_uid: 3, message: '', status: 1, created_at: 1 };
const group: FriendGroupDTO = { id: 1, name: '好友', sort_order: 1 };
const member: MemberDTO = { uid: 2, role: 0, last_read_seq: 0, join_time: 1 };

describe('chatReducer', () => {
  it('初始化当前用户缓存并支持通用 setField 兼容层', () => {
    const state = createInitialChatState(me);

    expect(state.userCache).toEqual({ 1: me });
    expect(chatReducer(state, { type: 'setField', key: 'selectedID', value: 9 }).selectedID).toBe(9);
  });

  it('加载基础数据时过滤隐藏会话、保留全部申请并只展示待处理申请', () => {
    const state = chatReducer(createInitialChatState(me), {
      type: 'baseLoaded',
      conversations: [conv(1), conv(2)],
      friends: [friend],
      incoming: [pendingIncoming],
      outgoing: [doneOutgoing],
      friendGroups: [group],
      hiddenIDs: new Set([2]),
    });

    expect(state.conversations.map((item) => item.conversation_id)).toEqual([1]);
    expect(state.requests).toEqual([pendingIncoming]);
    expect(state.outgoingReqs).toEqual([]);
    expect(state.allOutgoingReqs).toEqual([doneOutgoing]);
    expect(state.onlineMap).toEqual({ 2: false });
  });

  it('水合会话元信息时合并成员、详情、用户和预览', () => {
    const state = chatReducer(createInitialChatState(me), {
      type: 'hydrateChatMeta',
      members: { 1: [member] },
      details: { 1: detail(1, 2) },
      users: { 2: { ...me, id: 2, username: 'u2', nickname: 'U2' } },
      previews: { 1: 'hello' },
      chats: [conv(1)],
    });

    expect(state.members[1]).toEqual([member]);
    expect(state.details[1]?.type).toBe(2);
    expect(state.userCache[2]?.nickname).toBe('U2');
    expect(state.lastMsgMap[1]).toBe('hello');
  });

  it('打开、已读、隐藏会话都由 reducer 统一处理', () => {
    const base = { ...createInitialChatState(me), conversations: [conv(1, { unread_count: 3 }), conv(2)], selectedID: 1 };
    const opened = chatReducer(base, { type: 'openConversation', conversationID: 1 });
    const hidden = chatReducer(opened, { type: 'hideConversation', conversationID: 1 });

    expect(opened.conversations[0].unread_count).toBe(0);
    expect(opened.selectedID).toBe(1);
    expect(hidden.selectedID).toBeNull();
    expect(hidden.conversations.map((item) => item.conversation_id)).toEqual([2]);
  });

  it('收到消息时更新消息列表、预览、未读和输入态', () => {
    const state = chatReducer({ ...createInitialChatState(me), conversations: [conv(1)] }, {
      type: 'incomingMessage',
      message: message(1, 1),
      selectedID: null,
      currentUID: 1,
    });

    expect(state.messages[1]).toEqual([message(1, 1)]);
    expect(state.lastMsgMap[1]).toBe('message-1');
    expect(state.typing[1]).toBe('');
    expect(state.conversations[0].unread_count).toBe(1);
  });

  it('加载消息时过滤本地删除、合并分页并维护 hasMore 和预览', () => {
    const firstPage = chatReducer(createInitialChatState(me), {
      type: 'messagesLoaded',
      conversationID: 1,
      rawMessages: [message(2), message(1)],
      beforeSeq: 0,
      pageSize: 2,
      deletedIDs: new Set([2]),
    });
    const nextPage = chatReducer(firstPage, {
      type: 'messagesLoaded',
      conversationID: 1,
      rawMessages: [message(0)],
      beforeSeq: 1,
      pageSize: 2,
      deletedIDs: new Set(),
    });

    expect(firstPage.messages[1].map((item) => item.id)).toEqual([1]);
    expect(firstPage.lastMsgMap[1]).toBe('message-1');
    expect(firstPage.hasMore[1]).toBe(true);
    expect(nextPage.messages[1].map((item) => item.id)).toEqual([0, 1]);
    expect(nextPage.hasMore[1]).toBe(false);
  });

  it('本地删除消息时同步更新列表和会话预览', () => {
    const state = {
      ...createInitialChatState(me),
      messages: { 1: [message(1), message(2)] },
      lastMsgMap: { 1: 'message-2' },
    };
    const next = chatReducer(state, { type: 'deleteLocalMessage', message: message(2) });

    expect(next.messages[1].map((item) => item.id)).toEqual([1]);
    expect(next.lastMsgMap[1]).toBe('message-1');
  });
});
