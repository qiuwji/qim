import { api } from '@/api/http';
import { currentSecond } from '@/utils';
import type { MessageDTO } from '@/api/types';
import type { ChatStoreDeps } from '../types';
import { MSG_TYPE_TEXT, visibleMessages } from '../models/messageModel';

export function createMessageActions(d: ChatStoreDeps) {
  async function sendText(text: string, mentionUIDs?: number[], mentionAll?: boolean) {
    if (!d.selectedID || !text.trim()) return;
    try {
      d.sendConversationMessage(d.selectedID, text, MSG_TYPE_TEXT, d.replyTo?.id ?? 0, mentionUIDs, mentionAll);
      d.setReplyTo(null);
    } catch (err) {
      d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '发送失败' });
    }
  }

  async function sendImage(file: File) {
    if (!d.selectedID) return;
    try {
      const uploaded = await api.uploadImage(file);
      const clientID = d.wsRef.current.sendMessage({ conversation_id: d.selectedID, content: uploaded.url, msg_type: 2, reply_to: d.replyTo?.id ?? 0 });
      const optimistic: MessageDTO = { id: Date.now(), conversation_id: d.selectedID, seq: Number.MAX_SAFE_INTEGER, sender_id: d.user.id, msg_type: 2, content: uploaded.url, reply_to: d.replyTo?.id ?? 0, client_id: clientID, created_at: currentSecond() };
      d.applyIncomingMessage(optimistic);
      d.setReplyTo(null);
    } catch (err) {
      d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '图片发送失败' });
    }
  }

  function doRevoke(msg: MessageDTO) {
    if (!d.selectedID) return;
    d.setModal({ type: 'confirm', title: '撤回消息', text: '确认撤回这条消息？', onConfirm: () => {
      try { d.wsRef.current.revokeMessage(d.selectedID!, msg.id); } catch { d.setNotice({ kind: 'error', text: '撤回失败' }); }
    } });
  }

  function doDelete(msg: MessageDTO) {
    d.setModal({ type: 'confirm', title: '删除消息', text: '确认删除这条消息？删除只会在本地生效。', danger: true, onConfirm: () => {
      d.rememberDeletedMessage(msg);
      d.deleteLocalMessage(msg);
    } });
  }

  function doForward(msg: MessageDTO) {
    d.setModal({ type: 'conversation-picker', title: '转发消息', conversations: d.conversations, details: d.details, members: d.members, userCache: d.userCache, onlineMap: d.onlineMap, currentUID: d.user.id, lastMsgMap: d.lastMsgMap, mentionMap: d.mentionMap, onConfirm: (cid) => {
      try { d.wsRef.current.sendMessage({ conversation_id: cid, content: `[转发] ${msg.content}`, msg_type: msg.msg_type }); d.setNotice({ kind: 'ok', text: '已转发' }); }
      catch { d.setNotice({ kind: 'error', text: '转发失败' }); }
    } });
  }

  async function doChatSearch() {
    if (!d.selectedID || !d.chatSearch.trim()) return;
    try { const list = await api.searchMessages(d.selectedID, d.chatSearch.trim()); d.setChatSearchResult(visibleMessages(list, d.deletedMessageIDs.current).filter((m) => !m.revoked)); }
    catch { d.setChatSearchResult([]); }
  }

  return { sendText, sendImage, doRevoke, doDelete, doForward, doChatSearch };
}
