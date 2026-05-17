import { describe, expect, it } from 'vitest';
import type { FriendDTO, FriendGroupDTO, FriendRequestDTO, UserDTO } from '@/api/types';
import {
  filterFriendsByKeyword,
  friendDisplayName,
  friendRequestTimeline,
  groupFriendsByGroup,
  matchFriendKeyword,
  sortFriendGroups,
  userFromCache,
} from '../models/contactViewModel';

const userCache: Record<number, UserDTO> = {
  2: { id: 2, username: 'alice', nickname: 'Alice', avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 },
  3: { id: 3, username: 'bob', nickname: 'Bob', avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 },
};

const friends: FriendDTO[] = [
  { id: 1, friend_uid: 2, remark: '产品', group_id: 10, created_at: 1 },
  { id: 2, friend_uid: 3, remark: '', group_id: 0, created_at: 1 },
];

describe('contactViewModel', () => {
  it('从缓存生成用户，缺失时生成兜底用户', () => {
    expect(userFromCache(2, userCache).nickname).toBe('Alice');
    expect(userFromCache(9, userCache)).toMatchObject({ id: 9, nickname: '用户 9' });
  });

  it('统一好友展示名和分组', () => {
    expect(friendDisplayName(friends[0], userCache)).toBe('产品');
    expect(friendDisplayName(friends[1], userCache)).toBe('Bob');
    expect(groupFriendsByGroup(friends)).toEqual({ 0: [friends[1]], 10: [friends[0]] });
  });

  it('排序好友分组不修改原数组', () => {
    const groups: FriendGroupDTO[] = [
      { id: 2, name: 'B', sort_order: 2 },
      { id: 1, name: 'A', sort_order: 1 },
    ];

    expect(sortFriendGroups(groups).map((group) => group.id)).toEqual([1, 2]);
    expect(groups.map((group) => group.id)).toEqual([2, 1]);
  });

  it('按备注、昵称、用户名匹配好友搜索', () => {
    expect(matchFriendKeyword(friends[0], '产品', userCache)).toBe(true);
    expect(matchFriendKeyword(friends[1], 'ali', userCache)).toBe(false);
    expect(matchFriendKeyword(friends[1], 'bob', userCache)).toBe(true);
    expect(filterFriendsByKeyword(groupFriendsByGroup(friends), '产品', userCache)).toEqual({ 10: [friends[0]] });
  });

  it('合并好友申请并按时间倒序排列', () => {
    const incoming: FriendRequestDTO[] = [{ id: 1, from_uid: 2, to_uid: 1, message: '', status: 0, created_at: 10 }];
    const outgoing: FriendRequestDTO[] = [{ id: 2, from_uid: 1, to_uid: 3, message: '', status: 0, created_at: 20 }];

    expect(friendRequestTimeline(incoming, outgoing)).toEqual([
      { type: 'outgoing', req: outgoing[0], uid: 3 },
      { type: 'incoming', req: incoming[0], uid: 2 },
    ]);
  });
});
