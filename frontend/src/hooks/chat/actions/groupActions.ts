import { api } from '@/api/http';
import { currentSecond, displayName } from '@/utils';
import type { ChatStoreDeps } from '../types';
import { MSG_TYPE_SYSTEM } from '../models/messageModel';

export function createGroupActions(d: ChatStoreDeps) {
  function createGroup() {
    if (!d.friends.length) { d.setNotice({ kind: 'info', text: '暂无好友，先添加好友后再建群' }); return; }
    d.setModal({
      type: 'friend-picker', title: '创建群聊', friends: d.friends, userCache: d.userCache, requireGroupName: true,
      onConfirm: async ({ name, usernames }) => {
        if (!name) return;
        try {
          const conv = await api.createGroupChatByUsernames({ name, usernames });
          d.setDetails((p) => ({ ...p, [conv.id]: conv }));
          d.setConversations((p) => [{ conversation_id: conv.id, is_pinned: false, is_muted: false, unread_count: 0, last_msg_at: conv.created_at }, ...p]);
          d.setSelectedID(conv.id); d.setDetailOpen(false); d.setTab('chats'); d.setMobilePane('chat');
        } catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '建群失败' }); }
      },
    });
  }

  function renameGroup() {
    if (!d.selectedID) return;
    const cur = d.details[d.selectedID]?.name ?? '';
    d.setModal({ type: 'prompt', title: '修改群名称', fields: [{ key: 'name', label: '群聊名称', defaultValue: cur }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name || name === cur) return;
      try { await api.updateGroupInfo(d.selectedID!, { name }); d.setDetails((p) => ({ ...p, [d.selectedID!]: { ...p[d.selectedID!], name } })); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改群名称失败' }); }
    } });
  }

  function inviteMember() {
    if (!d.selectedID) return;
    const existingUIDs = (d.members[d.selectedID] ?? []).map((item) => item.uid);
    const availableFriends = d.friends.filter((friend) => !existingUIDs.includes(friend.friend_uid));
    if (!availableFriends.length) { d.setNotice({ kind: 'info', text: '暂无可邀请的好友' }); return; }
    d.setModal({ type: 'friend-picker', title: '邀请成员', friends: availableFriends, userCache: d.userCache, excludeUIDs: existingUIDs, onConfirm: async ({ usernames }) => {
      try { await Promise.all(usernames.map((username: string) => api.addMemberByUsername(d.selectedID!, username, 0))); await d.loadChatMembers(d.selectedID!, true); d.setNotice({ kind: 'ok', text: '已发送入群操作' }); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '邀请成员失败' }); }
    } });
  }

  function removeMember(uid: number) {
    if (!d.selectedID) return;
    const name = displayName(uid, d.userCache, d.friendMap);
    d.setModal({ type: 'confirm', title: '移除成员', text: `确认移除 ${name}？`, danger: true, onConfirm: async () => {
      try { await api.removeMember(d.selectedID!, uid); await d.loadChatMembers(d.selectedID!, true); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '移除成员失败' }); }
    } });
  }

  function leaveCurrentGroup() {
    if (!d.selectedID) return;
    d.setModal({ type: 'confirm', title: '退出群聊', text: '确认退出这个群聊？', danger: true, onConfirm: async () => {
      try { await api.leaveGroup(d.selectedID!); d.setConversations((p) => p.filter((i) => i.conversation_id !== d.selectedID)); d.setSelectedID(null); d.setDetailOpen(false); d.setMobilePane('list'); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '退群失败' }); }
    } });
  }

  function dissolveCurrentGroup() {
    if (!d.selectedID) return;
    const id = d.selectedID;
    d.setModal({ type: 'confirm', title: '解散群聊', text: '确认解散这个群聊？该操作不可恢复。', danger: true, onConfirm: async () => {
      try {
        await api.dissolveGroup(id);
        const now = Date.now();
        d.applyIncomingMessage({ id: now, conversation_id: id, seq: Number.MAX_SAFE_INTEGER, sender_id: d.user.id, msg_type: MSG_TYPE_SYSTEM, content: '群聊已解散', reply_to: 0, client_id: `system-dissolve-${id}-${now}`, created_at: currentSecond() });
        d.setNotice({ kind: 'info', text: '群聊已解散' });
      }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '解散群聊失败' }); }
    } });
  }

  function setMemberRole(uid: number, role: number) {
    if (!d.selectedID) return;
    const name = displayName(uid, d.userCache, d.friendMap);
    d.setModal({ type: 'confirm', title: '设置角色', text: `确认将 ${name} 设为${role === 1 ? '管理员' : '普通成员'}？`, onConfirm: async () => {
      try { await api.setMemberRole(d.selectedID!, uid, role); await d.loadChatMembers(d.selectedID!, true); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '设置角色失败' }); }
    } });
  }

  function transferOwner(uid: number) {
    if (!d.selectedID) return;
    const name = displayName(uid, d.userCache, d.friendMap);
    d.setModal({ type: 'confirm', title: '转让群主', text: `确认将群主转让给 ${name}？此操作不可撤销。`, danger: true, onConfirm: async () => {
      try { await api.transferOwner(d.selectedID!, uid); await d.loadChatMembers(d.selectedID!, true); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '转让群主失败' }); }
    } });
  }

  function uploadGroupAvatar(file: File) {
    if (!d.selectedID) return;
    (async () => {
      try {
        const u = await api.uploadImage(file);
        await api.updateGroupInfo(d.selectedID!, { avatar: u.url });
        d.setDetails((p) => ({ ...p, [d.selectedID!]: { ...p[d.selectedID!], avatar: u.url } }));
        d.setNotice({ kind: 'ok', text: '群头像已更新' });
      } catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '上传群头像失败' }); }
    })();
  }

  function setMemberLimit() {
    if (!d.selectedID) return;
    const cur = d.details[d.selectedID]?.member_limit ?? 0;
    d.setModal({ type: 'prompt', title: '成员上限', fields: [{ key: 'limit', label: '成员上限', defaultValue: String(cur), placeholder: '0 表示不限' }], onConfirm: async (v) => {
      const limit = Number(v.limit); if (isNaN(limit)) return;
      try { await api.updateGroupInfo(d.selectedID!, { member_limit: limit }); d.setDetails((p) => ({ ...p, [d.selectedID!]: { ...p[d.selectedID!], member_limit: limit } })); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '设置上限失败' }); }
    } });
  }

  return { createGroup, renameGroup, inviteMember, removeMember, leaveCurrentGroup, dissolveCurrentGroup, setMemberRole, transferOwner, uploadGroupAvatar, setMemberLimit };
}
