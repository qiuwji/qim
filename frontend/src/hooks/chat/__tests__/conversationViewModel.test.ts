import { describe, expect, it } from 'vitest';
import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import {
  canManageGroup,
  conversationAvatarUser,
  conversationMemberText,
  conversationSubtitle,
  fallbackUser,
  getConversationPeer,
  isGroupConversation,
  isPrivateConversation,
  memberDisplayUser,
} from '../models/conversationViewModel';

const privateDetail: ConversationDTO = {
  id: 1,
  type: 1,
  name: 'private',
  avatar: '',
  owner_id: 0,
  member_count: 2,
  member_limit: 2,
  max_seq: 0,
  created_at: 1,
};

const groupDetail: ConversationDTO = {
  id: 2,
  type: 2,
  name: 'group',
  avatar: 'group.png',
  owner_id: 1,
  member_count: 3,
  member_limit: 500,
  max_seq: 0,
  created_at: 1,
};

const conversation: UserConvDTO = {
  conversation_id: 2,
  is_pinned: false,
  is_muted: false,
  unread_count: 0,
  last_msg_at: 1,
};

const members: MemberDTO[] = [
  { uid: 1, role: 2, last_read_seq: 0, join_time: 1 },
  { uid: 3, role: 0, last_read_seq: 0, join_time: 1 },
];

const userCache: Record<number, UserDTO> = {
  3: { id: 3, username: 'tom', nickname: 'Tom', avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 },
};

const friendMap: Record<number, FriendDTO> = {
  3: { id: 1, friend_uid: 3, remark: '老友', group_id: 0, created_at: 1 },
};

describe('conversationViewModel', () => {
  it('按 conv.type 判断会话类型，不依赖成员数量', () => {
    expect(isPrivateConversation(privateDetail)).toBe(true);
    expect(isPrivateConversation(groupDetail)).toBe(false);
    expect(isGroupConversation(groupDetail)).toBe(true);
  });

  it('生成兜底用户和会话头像用户', () => {
    expect(fallbackUser(9).nickname).toBe('用户 9');
    expect(conversationAvatarUser(conversation, groupDetail)).toMatchObject({
      id: 2,
      nickname: 'group',
      avatar: 'group.png',
    });
  });

  it('统一成员展示用户和本人文案', () => {
    expect(memberDisplayUser(1, 1, {}, friendMap).nickname).toBe('我');
    expect(memberDisplayUser(3, 1, userCache, friendMap).nickname).toBe('Tom');
  });

  it('生成私聊、群聊和输入中副标题', () => {
    expect(getConversationPeer(members, 1)?.uid).toBe(3);
    expect(conversationSubtitle({ detail: privateDetail, members, currentUID: 1, userCache, friendMap, onlineMap: { 3: true } })).toBe('老友 · 在线');
    expect(conversationSubtitle({ detail: groupDetail, members, currentUID: 1, userCache, friendMap, onlineMap: {}, typingText: '正在输入...' })).toBe('正在输入...');
    expect(conversationSubtitle({ detail: groupDetail, members, currentUID: 1, userCache, friendMap, onlineMap: {} })).toBe('2 位成员');
  });

  it('统一成员数量文案和群管理权限', () => {
    expect(conversationMemberText(groupDetail, 3)).toBe('3 位成员');
    expect(conversationMemberText(privateDetail, 2)).toBe('私聊');
    expect(canManageGroup(groupDetail, members, 1)).toEqual({ isGroup: true, isOwner: true, isAdmin: true });
    expect(canManageGroup(groupDetail, members, 3)).toEqual({ isGroup: true, isOwner: false, isAdmin: false });
    expect(canManageGroup(privateDetail, members, 1)).toEqual({ isGroup: false, isOwner: false, isAdmin: false });
  });
});
