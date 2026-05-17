import { api } from '@/api/http';
import { displayName } from '@/utils';
import type { ChatStoreDeps } from './types';

const FRIEND_ACCEPTED_SYSTEM_TEXT = '我们的好友申请已经通过了，可以继续聊天了';
const FRIEND_ACCEPTED_DEFAULT_TEXT = '我们的好友申请通过了~';
const MSG_TYPE_SYSTEM = 3;
const MSG_TYPE_TEXT = 1;

export function createFriendActions(d: ChatStoreDeps) {
  function addFriendByUsername() {
    d.setModal({
      type: 'prompt', title: '添加好友',
      fields: [{ key: 'username', label: '账号', placeholder: '输入对方注册账号' }],
      onConfirm: async (v) => {
        const username = v.username?.trim();
        if (!username) return;
        if (username === d.user.username) { d.setNotice({ kind: 'error', text: '不能添加自己为好友' }); return; }
        try { await api.sendFriendRequestByUsername(username, '你好'); d.setNotice({ kind: 'ok', text: '好友申请已发送' }); await d.refreshBase(); }
        catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '添加好友失败' }); }
      },
    });
  }

  async function acceptFriendRequest(req: import('../../api/types').FriendRequestDTO) {
    await api.handleFriendRequest(req.id, 'accept');
    await d.refreshBase();
    const cid = await d.ensurePrivateConversation(req.from_uid);
    d.openConversation(cid);
    d.setReplyTo(null);
    try {
      d.sendConversationMessage(cid, FRIEND_ACCEPTED_SYSTEM_TEXT, MSG_TYPE_SYSTEM, 0);
      d.sendConversationMessage(cid, FRIEND_ACCEPTED_DEFAULT_TEXT, MSG_TYPE_TEXT, 0);
      d.setNotice({ kind: 'ok', text: '已同意好友申请' });
    } catch { d.setNotice({ kind: 'error', text: '已同意好友申请，但自动消息发送失败' }); }
  }

  async function handleRequest(reqID: number, action: 'accept' | 'reject') {
    try {
      const req = d.friends.length ? undefined : undefined;
      const found = (await api.incomingRequests()).find((item) => item.id === reqID);
      if (action === 'accept' && found) { await acceptFriendRequest(found); return; }
      await api.handleFriendRequest(reqID, action);
      await d.refreshBase();
    } catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '处理申请失败' }); }
  }

  function deleteFriend(uid: number) {
    d.setModal({ type: 'confirm', title: '删除好友', text: '确认删除该好友？', danger: true, onConfirm: async () => {
      try { await api.deleteFriend(uid); await d.refreshBase(); d.setNotice({ kind: 'ok', text: '已删除好友' }); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '删除好友失败' }); }
    } });
  }

  function updateFriendRemark(uid: number, cur: string) {
    d.setModal({ type: 'prompt', title: '修改备注', fields: [{ key: 'remark', label: '好友备注', defaultValue: cur }], onConfirm: async (v) => {
      try { await api.updateFriendRemark(uid, v.remark ?? ''); await d.refreshBase(); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改备注失败' }); }
    } });
  }

  function createFriendGroup() {
    d.setModal({ type: 'prompt', title: '新建分组', fields: [{ key: 'name', label: '分组名称', placeholder: '输入分组名' }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name) return;
      try { await api.createFriendGroup(name); await d.refreshBase(); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '创建分组失败' }); }
    } });
  }

  function renameFriendGroup(gid: number, cur: string) {
    d.setModal({ type: 'prompt', title: '重命名分组', fields: [{ key: 'name', label: '分组名称', defaultValue: cur }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name) return;
      try { await api.renameFriendGroup(gid, name); await d.refreshBase(); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '重命名失败' }); }
    } });
  }

  function deleteFriendGroup(gid: number) {
    d.setModal({ type: 'confirm', title: '删除分组', text: '删除分组后好友将移至默认分组', danger: true, onConfirm: async () => {
      try { await api.deleteFriendGroup(gid); await d.refreshBase(); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '删除分组失败' }); }
    } });
  }

  async function addFriendByUser(target: import('../../api/types').UserDTO) {
    if (target.id === d.user.id) return;
    try { await api.sendFriendRequestByUsername(target.username, '你好'); d.setNotice({ kind: 'ok', text: '好友申请已发送' }); }
    catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '添加好友失败' }); }
  }

  async function moveFriendGroup(friendUID: number, groupID: number) {
    try { await api.moveFriendGroup(friendUID, groupID); await d.refreshBase(); d.setNotice({ kind: 'ok', text: '已移动好友分组' }); }
    catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '移动分组失败' }); }
  }

  return { addFriendByUsername, addFriendByUser, handleRequest, deleteFriend, updateFriendRemark, createFriendGroup, renameFriendGroup, deleteFriendGroup, moveFriendGroup };
}
