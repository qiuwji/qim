import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ConversationDTO, MessageDTO, UserConvDTO } from '@/api/types';
import {
  applyIncomingConversation,
  decrementConversationUnread,
  forgetHiddenConversationID,
  groupConversations,
  hiddenConversationStorageKey,
  markConversationReadLocally,
  persistHiddenConversationID,
  readHiddenConversationIDs,
  sortConversations,
  visibleConversations,
} from '../models/conversationModel';

function createStorage() {
  const store = new Map<string, string>();
  return {
    getItem: vi.fn((key: string) => store.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => { store.set(key, value); }),
    removeItem: vi.fn((key: string) => { store.delete(key); }),
    clear: vi.fn(() => { store.clear(); }),
  };
}

function conv(id: number, overrides: Partial<UserConvDTO> = {}): UserConvDTO {
  return {
    conversation_id: id,
    is_pinned: false,
    is_muted: false,
    unread_count: 0,
    last_msg_at: id,
    ...overrides,
  };
}

function detail(id: number, type: 1 | 2): ConversationDTO {
  return {
    id,
    type,
    name: type === 2 ? `group-${id}` : `private-${id}`,
    avatar: '',
    owner_id: 0,
    member_count: type === 2 ? 2 : 2,
    member_limit: type === 2 ? 0 : 2,
    max_seq: 0,
    created_at: 1,
  };
}

function msg(conversationID: number, overrides: Partial<MessageDTO> = {}): MessageDTO {
  return {
    id: 100 + conversationID,
    conversation_id: conversationID,
    seq: 1,
    sender_id: 2,
    msg_type: 1,
    content: 'hello',
    reply_to: 0,
    client_id: '',
    created_at: 1000,
    ...overrides,
  };
}

describe('conversationModel', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', createStorage());
  });

  it('按置顶优先和最后消息时间排序', () => {
    const list = [
      conv(1, { last_msg_at: 10 }),
      conv(2, { is_pinned: true, last_msg_at: 1 }),
      conv(3, { is_pinned: true, last_msg_at: 20 }),
    ];

    expect(sortConversations(list).map((item) => item.conversation_id)).toEqual([3, 2, 1]);
    expect(list.map((item) => item.conversation_id)).toEqual([1, 2, 3]);
  });

  it('只按会话详情 type 筛出群聊，不用成员数量猜类型', () => {
    const list = [conv(1), conv(2), conv(3)];
    const details = {
      1: detail(1, 1),
      2: detail(2, 2),
    };

    expect(groupConversations(list, details).map((item) => item.conversation_id)).toEqual([2]);
  });

  it('读写本地隐藏会话 ID 并过滤无效值', () => {
    const key = hiddenConversationStorageKey(123);
    localStorage.setItem(key, JSON.stringify([1, '2', 0, -1, 'x']));

    expect(readHiddenConversationIDs(key)).toEqual([1, 2]);

    const hidden = new Set<number>([1]);
    persistHiddenConversationID(key, hidden, 3);
    expect([...hidden]).toEqual([1, 3]);
    expect(JSON.parse(localStorage.getItem(key) ?? '[]')).toEqual([1, 3]);

    forgetHiddenConversationID(key, hidden, 1);
    expect([...hidden]).toEqual([3]);
    expect(JSON.parse(localStorage.getItem(key) ?? '[]')).toEqual([3]);
  });

  it('本地隐藏配置损坏时返回空数组', () => {
    const key = hiddenConversationStorageKey(123);
    localStorage.setItem(key, '{bad json');

    expect(readHiddenConversationIDs(key)).toEqual([]);
  });

  it('从聊天列表过滤本地隐藏会话', () => {
    expect(visibleConversations([conv(1), conv(2), conv(3)], new Set([2])).map((item) => item.conversation_id)).toEqual([1, 3]);
  });

  it('本地已读和撤回未读递减不会影响其他会话', () => {
    const list = [conv(1, { unread_count: 3 }), conv(2, { unread_count: 0 })];

    expect(markConversationReadLocally(list, 1).find((item) => item.conversation_id === 1)?.unread_count).toBe(0);
    expect(decrementConversationUnread(list, 1).find((item) => item.conversation_id === 1)?.unread_count).toBe(2);
    expect(decrementConversationUnread(list, 2).find((item) => item.conversation_id === 2)?.unread_count).toBe(0);
  });

  it('收到当前打开会话消息时更新预览时间但不增加未读', () => {
    const list = [conv(1, { unread_count: 5, last_msg_at: 10 })];

    expect(applyIncomingConversation(list, msg(1, { created_at: 20 }), 1)).toEqual([
      { ...list[0], unread_count: 0, last_msg_at: 20 },
    ]);
  });

  it('收到非当前会话消息时增加未读，未知会话会插入列表', () => {
    const list = [conv(1, { unread_count: 5, last_msg_at: 10 })];

    expect(applyIncomingConversation(list, msg(1, { created_at: 20 }), 2)[0]).toMatchObject({
      conversation_id: 1,
      unread_count: 6,
      last_msg_at: 20,
    });
    expect(applyIncomingConversation(list, msg(9, { created_at: 30 }), null)[0]).toMatchObject({
      conversation_id: 9,
      unread_count: 1,
      last_msg_at: 30,
    });
  });
});
