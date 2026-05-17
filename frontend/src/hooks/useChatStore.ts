import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, clearSession, getSavedUser, getToken, saveSession } from '../api/http';
import { RealtimeClient } from '../api/ws';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO, WsResponse } from '../api/types';
import { currentSecond, displayName, upsertMessage, validatePassword } from '../utils';
import type { ContextMenu, MainTab, MobilePane, ModalState, Notice } from '../types';
import {
  applyIncomingConversation,
  groupConversations as filterGroupConversations,
  markConversationReadLocally,
  sortConversations,
} from './chat/conversationModel';
import { buildFriendMap, collectMissingFriendUserIDs, ensureFriendOnlineEntries } from './chat/friendModel';
import {
  applyMessagePreview,
  applyPreviewTexts,
  deletedMessageStorageKey,
  latestVisibleMessage,
  mergeLoadedMessages,
  messagePreviewText,
  MESSAGE_PAGE_SIZE,
  MSG_TYPE_SYSTEM,
  MSG_TYPE_TEXT,
  persistDeletedMessageID,
  readDeletedMessageIDs,
  removeMessageByID,
  visibleMessages,
} from './chat/messageModel';
import { handleRealtimeMessage } from './chat/realtimeModel';

const FRIEND_ACCEPTED_SYSTEM_TEXT = '我们的好友申请已经通过了，可以继续聊天了';
const FRIEND_ACCEPTED_DEFAULT_TEXT = '我们的好友申请通过了~';

export function useChatStore(user: UserDTO, onUserChange: (u: UserDTO) => void) {
  const wsRef = useRef(new RealtimeClient());
  const selectedIDRef = useRef<number | null>(null);
  const messagesRef = useRef<Record<number, MessageDTO[]>>({});
  const realtimeHandlerRef = useRef<(msg: WsResponse) => void>(() => undefined);
  const typingTimers = useRef<Record<number, ReturnType<typeof setTimeout>>>({});
  const deletedStorageKey = deletedMessageStorageKey(user.id);
  const deletedMessageIDs = useRef<Set<number>>(new Set(readDeletedMessageIDs(deletedStorageKey)));

  const [tab, setTab] = useState<MainTab>('chats');
  const [mobilePane, setMobilePane] = useState<MobilePane>('list');
  const [conversations, setConversations] = useState<UserConvDTO[]>([]);
  const [details, setDetails] = useState<Record<number, ConversationDTO>>({});
  const [userCache, setUserCache] = useState<Record<number, UserDTO>>({ [user.id]: user });
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [messages, setMessages] = useState<Record<number, MessageDTO[]>>({});
  const [lastMsgMap, setLastMsgMap] = useState<Record<number, string>>({});
  const [hasMore, setHasMore] = useState<Record<number, boolean>>({});
  const [friends, setFriends] = useState<FriendDTO[]>([]);
  const [friendGroups, setFriendGroups] = useState<FriendGroupDTO[]>([]);
  const [requests, setRequests] = useState<FriendRequestDTO[]>([]);
  const [outgoingReqs, setOutgoingReqs] = useState<FriendRequestDTO[]>([]);
  const [members, setMembers] = useState<Record<number, MemberDTO[]>>({});
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResult, setSearchResult] = useState<UserDTO[]>([]);
  const [typing, setTyping] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [modal, setModal] = useState<ModalState>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenu>(null);
  const [replyTo, setReplyTo] = useState<MessageDTO | null>(null);
  const [chatSearch, setChatSearch] = useState('');
  const [chatSearchResult, setChatSearchResult] = useState<MessageDTO[]>([]);
  const [notice, setNotice] = useState<Notice>(null);
  const [viewingUser, setViewingUser] = useState<UserDTO | null>(null);
  const [hoverCard, setHoverCard] = useState<{ user: UserDTO; rect: DOMRect } | null>(null);
  const [onlineMap, setOnlineMap] = useState<Record<number, boolean>>({});

  const selectedConv = conversations.find((item) => item.conversation_id === selectedID) ?? null;
  const sortedConversations = useMemo(() => sortConversations(conversations), [conversations]);
  const groupConversations = useMemo(() => filterGroupConversations(sortedConversations, details), [sortedConversations, details]);
  const unreadTotal = useMemo(() => conversations.reduce((s, c) => s + c.unread_count, 0), [conversations]);
  const friendMap = useMemo(() => buildFriendMap(friends), [friends]);

  useEffect(() => { selectedIDRef.current = selectedID; }, [selectedID]);
  useEffect(() => { messagesRef.current = messages; }, [messages]);

  useEffect(() => {
    if (unreadTotal > 0) document.title = `(${unreadTotal}) QIM`;
    else document.title = 'QIM';
  }, [unreadTotal]);

  const refreshBase = useCallback(async () => {
    setLoading(true);
    try {
      const [convList, friendList, incoming, outgoing, groups] = await Promise.all([
        api.chats(), api.friends(), api.incomingRequests(), api.outgoingRequests(), api.friendGroups(),
      ]);
      setConversations(convList);
      setFriends(friendList);
      setOnlineMap((prev) => ensureFriendOnlineEntries(friendList, prev));
      setRequests(incoming.filter((i) => i.status === 0));
      setOutgoingReqs(outgoing.filter((i) => i.status === 0));
      setFriendGroups(groups);
      await hydrateChatMeta(convList, friendList);
      await hydrateFriendUsers(friendList, incoming, outgoing);
      wsRef.current.requestOnlineFriends();
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '加载失败' });
    } finally {
      setLoading(false);
    }
  }, [user.id]);

  async function hydrateFriendUsers(friendList: FriendDTO[], incomingReqs: FriendRequestDTO[], outgoingReqs: FriendRequestDTO[]) {
    const missing = collectMissingFriendUserIDs(friendList, incomingReqs, outgoingReqs, userCache);
    if (!missing.length) return;
    const users = await Promise.all(missing.map((uid) => api.getUser(uid).catch(() => null)));
    const next: Record<number, UserDTO> = {};
    for (const u of users) if (u) next[u.id] = u;
    if (Object.keys(next).length) setUserCache((prev) => ({ ...prev, ...next }));
  }

  function rememberDeletedMessage(msg: MessageDTO) {
    persistDeletedMessageID(deletedStorageKey, deletedMessageIDs.current, msg.id);
  }

  function setLastMessagePreview(cid: number, msg: MessageDTO | undefined) {
    setLastMsgMap((prev) => applyMessagePreview(prev, cid, msg));
  }

  function markConversationRead(cid: number, seq = 0) {
    setConversations((p) => markConversationReadLocally(p, cid));
    void api.markRead(cid, seq).catch(() => undefined);
  }

  function openConversation(cid: number) {
    setSelectedID(cid);
    selectedIDRef.current = cid;
    setViewingUser(null);
    setDetailOpen(false);
    setTab('chats');
    setMobilePane('chat');
    markConversationRead(cid);
  }

  async function hydrateChatMeta(chatList: UserConvDTO[], friendList?: FriendDTO[]) {
    const nextM: Record<number, MemberDTO[]> = {};
    const nextD: Record<number, ConversationDTO> = {};
    const nextU: Record<number, UserDTO> = {};
    const nextL: Record<number, string | undefined> = {};
    await Promise.all(chatList.map(async (chat) => {
      const id = chat.conversation_id;
      const [ml, msgList] = await Promise.all([api.members(id).catch(() => []), api.messages(id, 0, MESSAGE_PAGE_SIZE).catch(() => [])]);
      nextM[id] = ml;
      nextL[id] = messagePreviewText(latestVisibleMessage(msgList, deletedMessageIDs.current));
      const peer = ml.length === 2 ? ml.find((m) => m.uid !== user.id) : undefined;
      if (peer) {
        const pu = await api.getUser(peer.uid).catch(() => null) ?? userCache[peer.uid];
        if (pu) {
          nextU[peer.uid] = pu;
          const peerName = friendList?.find((f) => f.friend_uid === peer.uid)?.remark || pu.nickname || pu.username;
          nextD[id] = { id, type: 1, name: peerName, avatar: pu.avatar, owner_id: 0, member_count: ml.length, member_limit: 2, max_seq: 0, created_at: chat.last_msg_at };
          return;
        }
      }
      nextD[id] = details[id] ?? { id, type: 2, name: `群聊 ${id}`, avatar: '', owner_id: 0, member_count: ml.length, member_limit: 0, max_seq: 0, created_at: chat.last_msg_at };
    }));
    setMembers((p) => ({ ...p, ...nextM }));
    setDetails((p) => ({ ...p, ...nextD }));
    if (Object.keys(nextU).length) setUserCache((p) => ({ ...p, ...nextU }));
    setLastMsgMap((p) => applyPreviewTexts(p, chatList, nextL));
  }

  useEffect(() => {
    const off = wsRef.current.on((msg) => realtimeHandlerRef.current(msg));
    wsRef.current.connect(() => { setOnlineMap({}); void refreshBase(); });
    void refreshBase();
    return () => { off(); wsRef.current.close(); };
  }, []);

  useEffect(() => { if (!selectedID) return; void loadMessages(selectedID); void loadChatMembers(selectedID); }, [selectedID]);
  useEffect(() => { if (!selectedID) return; const t = window.setInterval(() => void loadChatMembers(selectedID, true), 5000); return () => window.clearInterval(t); }, [selectedID]);

  function handleWsMessage(msg: WsResponse) {
    handleRealtimeMessage(msg, {
      currentUID: user.id,
      ws: wsRef.current,
      typingTimers,
      deletedMessageIDs,
      getMessages: () => messagesRef.current,
      getDisplayName: (uid) => displayName(uid, userCache, friendMap),
      applyIncomingMessage,
      setLastMessagePreview,
      setNotice,
      setMessages,
      setLastMsgMap,
      setConversations,
      setTyping,
      setOnlineMap,
      refreshBase,
    });
  }

  realtimeHandlerRef.current = handleWsMessage;

  function applyIncomingMessage(next: MessageDTO) {
    if (deletedMessageIDs.current.has(next.id)) return;
    setMessages((p) => ({ ...p, [next.conversation_id]: upsertMessage(p[next.conversation_id] ?? [], next) }));
    const text = messagePreviewText(next);
    if (text) setLastMsgMap((p) => ({ ...p, [next.conversation_id]: text }));
    if (next.sender_id !== user.id) setTyping((p) => ({ ...p, [next.conversation_id]: '' }));
    setConversations((p) => applyIncomingConversation(p, next, selectedIDRef.current));
    if (selectedIDRef.current === next.conversation_id && next.seq > 0) markConversationRead(next.conversation_id, next.seq);
    if (selectedIDRef.current !== next.conversation_id && !document.hasFocus()) {
      try { new Notification('QIM 新消息', { body: next.content.slice(0, 50) }); } catch {}
    }
  }

  async function loadMessages(cid: number, beforeSeq = 0) {
    try {
      const rawList = await api.messages(cid, beforeSeq, MESSAGE_PAGE_SIZE);
      const list = visibleMessages(rawList, deletedMessageIDs.current);
      const newest = list.length > 0 ? list[0] : undefined;
      setMessages((p) => {
        const existing = beforeSeq > 0 ? (p[cid] ?? []) : [];
        return { ...p, [cid]: mergeLoadedMessages(list, existing, beforeSeq) };
      });
      if (beforeSeq === 0) setLastMessagePreview(cid, newest);
      setHasMore((p) => ({ ...p, [cid]: rawList.length >= MESSAGE_PAGE_SIZE }));
      const maxSeq = newest?.seq ?? 0;
      if (maxSeq > 0 && beforeSeq === 0) markConversationRead(cid, maxSeq);
      if (beforeSeq === 0) {
        await loadChatMembers(cid, true);
      }
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '消息加载失败' });
    }
  }

  async function loadChatMembers(cid: number, force = false) {
    if (!force && members[cid]) return;
    try { const list = await api.members(cid); setMembers((p) => ({ ...p, [cid]: list })); }
    catch { setMembers((p) => ({ ...p, [cid]: [] })); }
  }

  function sendConversationMessage(conversationID: number, text: string, msgType = MSG_TYPE_TEXT, replyToID = msgType === MSG_TYPE_TEXT ? replyTo?.id ?? 0 : 0) {
    const content = text.trim();
    if (!content) return;
    const clientID = wsRef.current.sendMessage({ conversation_id: conversationID, content, msg_type: msgType, reply_to: replyToID });
    const optimistic: MessageDTO = {
      id: Date.now(),
      conversation_id: conversationID,
      seq: Number.MAX_SAFE_INTEGER,
      sender_id: user.id,
      msg_type: msgType,
      content,
      reply_to: replyToID,
      client_id: clientID,
      created_at: currentSecond(),
    };
    applyIncomingMessage(optimistic);
  }

  async function sendText(text: string) {
    if (!selectedID || !text.trim()) return;
    try {
      sendConversationMessage(selectedID, text, MSG_TYPE_TEXT);
      setReplyTo(null);
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '发送失败' });
    }
  }

  async function sendImage(file: File) {
    if (!selectedID) return;
    try {
      const uploaded = await api.uploadImage(file);
      const clientID = wsRef.current.sendMessage({ conversation_id: selectedID, content: uploaded.url, msg_type: 2, reply_to: replyTo?.id ?? 0 });
      const optimistic: MessageDTO = { id: Date.now(), conversation_id: selectedID, seq: Number.MAX_SAFE_INTEGER, sender_id: user.id, msg_type: 2, content: uploaded.url, reply_to: replyTo?.id ?? 0, client_id: clientID, created_at: currentSecond() };
      applyIncomingMessage(optimistic);
      setReplyTo(null);
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '图片发送失败' });
    }
  }

  async function searchUsers() {
    if (!searchKeyword.trim()) return;
    try { setSearchResult(await api.searchUsers(searchKeyword.trim())); }
    catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '搜索失败' }); }
  }

  async function ensurePrivateConversation(uid: number): Promise<number> {
    const existing = conversations.find((conv) => {
      if (details[conv.conversation_id]?.type !== 1) return false;
      return (members[conv.conversation_id] ?? []).some((m) => m.uid === uid);
    });
    if (existing) return existing.conversation_id;

    const conv = await api.createPrivateChat(uid);
    const [peer, memberList] = await Promise.all([
      userCache[uid] ? Promise.resolve(userCache[uid]) : api.getUser(uid).catch(() => null),
      api.members(conv.id).catch(() => []),
    ]);
    if (peer) setUserCache((p) => ({ ...p, [uid]: peer }));
    setDetails((p) => ({ ...p, [conv.id]: peer ? { ...conv, type: 1, name: peer.nickname || peer.username, avatar: peer.avatar } : conv }));
    setMembers((p) => ({ ...p, [conv.id]: memberList.length ? memberList : [{ uid: user.id, role: 1, last_read_seq: 0, join_time: conv.created_at }, { uid, role: 1, last_read_seq: 0, join_time: conv.created_at }] }));
    setConversations((p) => (p.some((i) => i.conversation_id === conv.id) ? p : [{ conversation_id: conv.id, is_pinned: false, is_muted: false, unread_count: 0, last_msg_at: conv.created_at }, ...p]));
    return conv.id;
  }

  async function startPrivate(uid: number) {
    if (!friendMap[uid]) {
      setNotice({ kind: 'error', text: '只能给好友发消息，请先添加好友' });
      return;
    }
    try {
      const cid = await ensurePrivateConversation(uid);
      openConversation(cid);
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '创建聊天失败' });
    }
  }

  async function acceptFriendRequest(req: FriendRequestDTO) {
    await api.handleFriendRequest(req.id, 'accept');
    await refreshBase();
    const cid = await ensurePrivateConversation(req.from_uid);
    openConversation(cid);
    setReplyTo(null);
    try {
      sendConversationMessage(cid, FRIEND_ACCEPTED_SYSTEM_TEXT, MSG_TYPE_SYSTEM, 0);
      sendConversationMessage(cid, FRIEND_ACCEPTED_DEFAULT_TEXT, MSG_TYPE_TEXT, 0);
      setNotice({ kind: 'ok', text: '已同意好友申请' });
    } catch {
      setNotice({ kind: 'error', text: '已同意好友申请，但自动消息发送失败' });
    }
  }

  function createGroup() {
    if (!friends.length) {
      setNotice({ kind: 'info', text: '暂无好友，先添加好友后再建群' });
      return;
    }
    setModal({
      type: 'friend-picker',
      title: '创建群聊',
      friends,
      userCache,
      requireGroupName: true,
      onConfirm: async ({ name, usernames }) => {
        if (!name) return;
        try {
          const conv = await api.createGroupChatByUsernames({ name, usernames });
          setDetails((p) => ({ ...p, [conv.id]: conv }));
          setConversations((p) => [{ conversation_id: conv.id, is_pinned: false, is_muted: false, unread_count: 0, last_msg_at: conv.created_at }, ...p]);
          setSelectedID(conv.id); setDetailOpen(false); setTab('chats'); setMobilePane('chat');
        } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '建群失败' }); }
      },
    });
  }

  function addFriendByUsername() {
    setModal({
      type: 'prompt', title: '添加好友',
      fields: [{ key: 'username', label: '账号', placeholder: '输入对方注册账号' }],
      onConfirm: async (v) => {
        const username = v.username?.trim();
        if (!username) return;
        if (username === user.username) {
          setNotice({ kind: 'error', text: '不能添加自己为好友' });
          return;
        }
        try {
          await api.sendFriendRequestByUsername(username, '你好');
          setNotice({ kind: 'ok', text: '好友申请已发送' });
          await refreshBase();
        } catch (err) {
          setNotice({ kind: 'error', text: err instanceof Error ? err.message : '添加好友失败' });
        }
      },
    });
  }

  async function handleRequest(reqID: number, action: 'accept' | 'reject') {
    try {
      const req = requests.find((item) => item.id === reqID);
      if (action === 'accept' && req) {
        await acceptFriendRequest(req);
        return;
      }
      await api.handleFriendRequest(reqID, action);
      await refreshBase();
    }
    catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '处理申请失败' }); }
  }

  async function uploadAvatar(file: File) {
    try {
      const u = await api.uploadImage(file);
      await api.updateProfile({ avatar: u.url });
      const next = { ...user, avatar: u.url };
      saveSession(getToken(), next);
      onUserChange(next);
      setUserCache((p) => ({ ...p, [user.id]: next }));
      setNotice({ kind: 'ok', text: '头像已更新' });
    } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '上传失败' }); }
  }

  async function togglePin(convID?: number) {
    const id = convID ?? selectedID;
    if (!id) return;
    let nextPinned = false;
    setConversations((prev) => {
      const conv = prev.find((c) => c.conversation_id === id);
      if (!conv) return prev;
      nextPinned = !conv.is_pinned;
      return prev.map((i) => (i.conversation_id === id ? { ...i, is_pinned: nextPinned } : i));
    });
    try { await api.pinChat(id, nextPinned); }
    catch (err) { setConversations((p) => p.map((i) => (i.conversation_id === id ? { ...i, is_pinned: !nextPinned } : i))); setNotice({ kind: 'error', text: err instanceof Error ? err.message : '置顶失败' }); }
  }

  async function toggleMute(convID?: number) {
    const id = convID ?? selectedID;
    if (!id) return;
    let nextMuted = false;
    setConversations((prev) => {
      const conv = prev.find((c) => c.conversation_id === id);
      if (!conv) return prev;
      nextMuted = !conv.is_muted;
      return prev.map((i) => (i.conversation_id === id ? { ...i, is_muted: nextMuted } : i));
    });
    try { await api.muteChat(id, nextMuted); }
    catch (err) { setConversations((p) => p.map((i) => (i.conversation_id === id ? { ...i, is_muted: !nextMuted } : i))); setNotice({ kind: 'error', text: err instanceof Error ? err.message : '免打扰设置失败' }); }
  }

  async function markAllRead() {
    try { await api.markAllRead(); setConversations((p) => p.map((i) => ({ ...i, unread_count: 0 }))); }
    catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '全部已读失败' }); }
  }

  function renameGroup() {
    if (!selectedID) return;
    const cur = details[selectedID]?.name ?? '';
    setModal({ type: 'prompt', title: '修改群名称', fields: [{ key: 'name', label: '群聊名称', defaultValue: cur }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name || name === cur) return;
      try { await api.updateGroupInfo(selectedID, { name }); setDetails((p) => ({ ...p, [selectedID]: { ...p[selectedID], name } })); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改群名称失败' }); }
    } });
  }

  function inviteMember() {
    if (!selectedID) return;
    const existingUIDs = (members[selectedID] ?? []).map((item) => item.uid);
    const availableFriends = friends.filter((friend) => !existingUIDs.includes(friend.friend_uid));
    if (!availableFriends.length) {
      setNotice({ kind: 'info', text: '暂无可邀请的好友' });
      return;
    }
    setModal({ type: 'friend-picker', title: '邀请成员', friends: availableFriends, userCache, excludeUIDs: existingUIDs, onConfirm: async ({ usernames }) => {
      try {
        await Promise.all(usernames.map((username) => api.addMemberByUsername(selectedID, username, 0)));
        await loadChatMembers(selectedID, true);
        setNotice({ kind: 'ok', text: '已发送入群操作' });
      }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '邀请成员失败' }); }
    } });
  }

  function removeMember(uid: number) {
    if (!selectedID) return;
    const name = displayName(uid, userCache, friendMap);
    setModal({ type: 'confirm', title: '移除成员', text: `确认移除 ${name}？`, danger: true, onConfirm: async () => {
      try { await api.removeMember(selectedID, uid); await loadChatMembers(selectedID, true); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '移除成员失败' }); }
    } });
  }

  function leaveCurrentGroup() {
    if (!selectedID) return;
    setModal({ type: 'confirm', title: '退出群聊', text: '确认退出这个群聊？', danger: true, onConfirm: async () => {
      try { await api.leaveGroup(selectedID); setConversations((p) => p.filter((i) => i.conversation_id !== selectedID)); setSelectedID(null); setDetailOpen(false); setMobilePane('list'); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '退群失败' }); }
    } });
  }

  function dissolveCurrentGroup() {
    if (!selectedID) return;
    setModal({ type: 'confirm', title: '解散群聊', text: '确认解散这个群聊？该操作不可恢复。', danger: true, onConfirm: async () => {
      try { await api.dissolveGroup(selectedID); setConversations((p) => p.filter((i) => i.conversation_id !== selectedID)); setSelectedID(null); setDetailOpen(false); setMobilePane('list'); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '解散群聊失败' }); }
    } });
  }

  function setMemberRole(uid: number, role: number) {
    if (!selectedID) return;
    const name = displayName(uid, userCache, friendMap);
    setModal({ type: 'confirm', title: '设置角色', text: `确认将 ${name} 设为${role === 1 ? '管理员' : '普通成员'}？`, onConfirm: async () => {
      try { await api.setMemberRole(selectedID, uid, role); await loadChatMembers(selectedID, true); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '设置角色失败' }); }
    } });
  }

  function transferOwner(uid: number) {
    if (!selectedID) return;
    const name = displayName(uid, userCache, friendMap);
    setModal({ type: 'confirm', title: '转让群主', text: `确认将群主转让给 ${name}？此操作不可撤销。`, danger: true, onConfirm: async () => {
      try { await api.transferOwner(selectedID, uid); await loadChatMembers(selectedID, true); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '转让群主失败' }); }
    } });
  }

  function uploadGroupAvatar(file: File) {
    if (!selectedID) return;
    (async () => {
      try {
        const u = await api.uploadImage(file);
        await api.updateGroupInfo(selectedID, { avatar: u.url });
        setDetails((p) => ({ ...p, [selectedID]: { ...p[selectedID], avatar: u.url } }));
        setNotice({ kind: 'ok', text: '群头像已更新' });
      } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '上传群头像失败' }); }
    })();
  }

  function setMemberLimit() {
    if (!selectedID) return;
    const cur = details[selectedID]?.member_limit ?? 0;
    setModal({ type: 'prompt', title: '成员上限', fields: [{ key: 'limit', label: '成员上限', defaultValue: String(cur), placeholder: '0 表示不限' }], onConfirm: async (v) => {
      const limit = Number(v.limit); if (isNaN(limit)) return;
      try { await api.updateGroupInfo(selectedID, { member_limit: limit }); setDetails((p) => ({ ...p, [selectedID]: { ...p[selectedID], member_limit: limit } })); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '设置上限失败' }); }
    } });
  }

  function deleteFriend(uid: number) {
    setModal({ type: 'confirm', title: '删除好友', text: '确认删除该好友？', danger: true, onConfirm: async () => {
      try { await api.deleteFriend(uid); await refreshBase(); setNotice({ kind: 'ok', text: '已删除好友' }); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '删除好友失败' }); }
    } });
  }

  function updateFriendRemark(uid: number, cur: string) {
    setModal({ type: 'prompt', title: '修改备注', fields: [{ key: 'remark', label: '好友备注', defaultValue: cur }], onConfirm: async (v) => {
      try { await api.updateFriendRemark(uid, v.remark ?? ''); await refreshBase(); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改备注失败' }); }
    } });
  }

  function createFriendGroup() {
    setModal({ type: 'prompt', title: '新建分组', fields: [{ key: 'name', label: '分组名称', placeholder: '输入分组名' }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name) return;
      try { await api.createFriendGroup(name); await refreshBase(); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '创建分组失败' }); }
    } });
  }

  function renameFriendGroup(gid: number, cur: string) {
    setModal({ type: 'prompt', title: '重命名分组', fields: [{ key: 'name', label: '分组名称', defaultValue: cur }], onConfirm: async (v) => {
      const name = v.name?.trim(); if (!name) return;
      try { await api.renameFriendGroup(gid, name); await refreshBase(); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '重命名失败' }); }
    } });
  }

  function deleteFriendGroup(gid: number) {
    setModal({ type: 'confirm', title: '删除分组', text: '删除分组后好友将移至默认分组', danger: true, onConfirm: async () => {
      try { await api.deleteFriendGroup(gid); await refreshBase(); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '删除分组失败' }); }
    } });
  }

  function updateProfile() {
    setModal({ type: 'prompt', title: '修改资料', fields: [{ key: 'nickname', label: '昵称', defaultValue: user.nickname }, { key: 'sign', label: '个性签名', defaultValue: user.sign }], onConfirm: async (v) => {
      try {
        await api.updateProfile({ nickname: v.nickname, sign: v.sign });
        const next = { ...user, nickname: v.nickname ?? user.nickname, sign: v.sign ?? user.sign };
        saveSession(getToken(), next);
        onUserChange(next);
        setUserCache((p) => ({ ...p, [user.id]: next }));
      } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改资料失败' }); }
    } });
  }

  function changePassword() {
    setModal({ type: 'prompt', title: '修改密码', fields: [{ key: 'old', label: '旧密码', placeholder: '输入旧密码' }, { key: 'new', label: '新密码', placeholder: '8-20位，含大小写字母和特殊字符' }], onConfirm: async (v) => {
      const pwdErr = validatePassword(v.new ?? '');
      if (pwdErr) { setNotice({ kind: 'error', text: pwdErr }); return; }
      try { await api.changePassword({ old_password: v.old ?? '', new_password: v.new ?? '' }); setNotice({ kind: 'ok', text: '密码已修改' }); }
      catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改密码失败' }); }
    } });
  }

  function doRevoke(msg: MessageDTO) {
    if (!selectedID) return;
    setModal({ type: 'confirm', title: '撤回消息', text: '确认撤回这条消息？', onConfirm: () => {
      try { wsRef.current.revokeMessage(selectedID, msg.id); } catch { setNotice({ kind: 'error', text: '撤回失败' }); }
    } });
  }

  function doDelete(msg: MessageDTO) {
    setModal({ type: 'confirm', title: '删除消息', text: '确认删除这条消息？删除只会在本地生效。', danger: true, onConfirm: () => {
      rememberDeletedMessage(msg);
      setMessages((p) => {
        const nextList = removeMessageByID(p[msg.conversation_id] ?? [], msg.id);
        const last = nextList[nextList.length - 1];
        setLastMessagePreview(msg.conversation_id, last);
        return { ...p, [msg.conversation_id]: nextList };
      });
    } });
  }

  function doForward(msg: MessageDTO) {
    setModal({ type: 'prompt', title: '转发消息', fields: [{ key: 'cid', label: '目标聊天 ID', placeholder: '输入聊天 ID' }], onConfirm: async (v) => {
      const cid = Number(v.cid); if (!cid) return;
      try { wsRef.current.sendMessage({ conversation_id: cid, content: `[转发] ${msg.content}`, msg_type: msg.msg_type }); setNotice({ kind: 'ok', text: '已转发' }); }
      catch { setNotice({ kind: 'error', text: '转发失败' }); }
    } });
  }

  async function doChatSearch() {
    if (!selectedID || !chatSearch.trim()) return;
    try { const list = await api.searchMessages(selectedID, chatSearch.trim()); setChatSearchResult(visibleMessages(list, deletedMessageIDs.current).filter((m) => !m.revoked)); }
    catch { setChatSearchResult([]); }
  }

  function selectChat(id: number) {
    if (selectedID === id) { setSelectedID(null); selectedIDRef.current = null; setDetailOpen(false); setMobilePane('list'); }
    else { openConversation(id); }
    setReplyTo(null); setChatSearchResult([]);
  }

  async function viewUserProfile(uid: number) {
    try {
      const u = userCache[uid] ?? (await api.getUser(uid));
      if (u) { setUserCache((p) => ({ ...p, [uid]: u })); setViewingUser(u); setMobilePane('chat'); }
    } catch { setNotice({ kind: 'error', text: '获取用户信息失败' }); }
  }

  function showHoverCard(uid: number, rect: DOMRect) {
    const u = userCache[uid];
    if (u) setHoverCard({ user: u, rect });
  }

  useEffect(() => { if ('Notification' in window && Notification.permission === 'default') Notification.requestPermission(); }, []);

  return {
    tab, setTab, mobilePane, setMobilePane,
    conversations, sortedConversations, groupConversations, details, userCache,
    selectedID, selectedConv, messages, lastMsgMap, hasMore, friends, friendGroups, friendMap,
    requests, outgoingReqs, members, searchKeyword, setSearchKeyword, searchResult,
    typing, loading, detailOpen, setDetailOpen, modal, setModal, contextMenu,
    setContextMenu, replyTo, setReplyTo, chatSearch, setChatSearch, chatSearchResult, setChatSearchResult,
    notice, setNotice, unreadTotal,
    viewingUser, setViewingUser, hoverCard, setHoverCard, onlineMap,
    refreshBase, loadMessages, loadChatMembers, sendText, sendImage, searchUsers,
    startPrivate, createGroup, addFriendByUsername, handleRequest, uploadAvatar, togglePin, toggleMute,
    markAllRead, renameGroup, inviteMember, removeMember, leaveCurrentGroup,
    dissolveCurrentGroup, setMemberRole, transferOwner, uploadGroupAvatar,
    setMemberLimit, deleteFriend, updateFriendRemark, createFriendGroup,
    renameFriendGroup, deleteFriendGroup, updateProfile, changePassword,
    doRevoke, doDelete, doForward, doChatSearch, selectChat,
    viewUserProfile, showHoverCard,
    wsRef,
  };
}

export type ChatStore = ReturnType<typeof useChatStore>;

export function useAuth() {
  const [user, setUser] = useState<UserDTO | null>(() => getSavedUser());
  const [notice, setNotice] = useState<Notice>(null);

  function handleLoggedIn(token: string, u: UserDTO) { saveSession(token, u); setUser(u); }
  function logout() { clearSession(); setUser(null); }

  return { user, setUser, notice, setNotice, handleLoggedIn, logout };
}
