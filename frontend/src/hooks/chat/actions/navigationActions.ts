import { api } from '@/api/http';
import type { ChatStoreDeps } from '../types';

export function createNavigationActions(d: ChatStoreDeps) {
  async function searchUsers(kw?: string) {
    const q = (kw ?? d.searchKeyword).trim();
    if (!q) {
      d.setSearchResult([]);
      return;
    }
    try {
      d.setSearchResult(await api.searchUsers(q));
    } catch (err) {
      d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '搜索失败' });
    }
  }

  async function startPrivate(uid: number) {
    if (!d.friendMap[uid]) {
      d.setNotice({ kind: 'error', text: '只能给好友发消息，请先添加好友' });
      return;
    }
    try {
      const cid = await d.ensurePrivateConversation(uid);
      d.openConversation(cid);
    } catch (err) {
      d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '创建聊天失败' });
    }
  }

  async function viewUserProfile(uid: number) {
    try {
      const user = d.userCache[uid] ?? (await api.getUser(uid));
      if (!user) return;
      d.setUserCache((prev) => ({ ...prev, [uid]: user }));
      d.setViewingUser(user);
      d.setViewingFriendRequests(false);
      d.setViewingGroupManage(false);
      d.setSelectedID(null);
      d.selectedIDRef.current = null;
      d.setMobilePane('chat');
    } catch {
      d.setNotice({ kind: 'error', text: '获取用户信息失败' });
    }
  }

  function viewFriendRequests() {
    d.setViewingFriendRequests(true);
    d.setViewingUser(null);
    d.setViewingGroupManage(false);
    d.setSelectedID(null);
    d.selectedIDRef.current = null;
    d.setMobilePane('chat');
  }

  function viewGroupManage() {
    d.setViewingGroupManage(true);
    d.setViewingUser(null);
    d.setViewingFriendRequests(false);
    d.setSelectedID(null);
    d.selectedIDRef.current = null;
    d.setMobilePane('chat');
  }

  function showHoverCard(uid: number, rect: DOMRect) {
    const user = d.userCache[uid];
    if (user) d.setHoverCard({ user, rect });
  }

  return { searchUsers, startPrivate, viewUserProfile, viewFriendRequests, viewGroupManage, showHoverCard };
}
