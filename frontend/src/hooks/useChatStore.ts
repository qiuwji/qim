import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, clearSession, getSavedUser, getToken, saveSession } from '@/api/http';
import { RealtimeClient } from '@/api/ws';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO, WsResponse } from '@/api/types';
import { currentSecond, displayName, upsertMessage } from '@/utils';
import type { ContextMenu, MainTab, MobilePane, ModalState, Notice } from '@/types';
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
  MSG_TYPE_TEXT,
  persistDeletedMessageID,
  readDeletedMessageIDs,
  removeMessageByID,
  visibleMessages,
} from './chat/messageModel';
import { handleRealtimeMessage } from './chat/realtimeModel';
import type { ChatStoreDeps } from './chat/types';
import { createMessageActions } from './chat/messageActions';
import { createConversationActions } from './chat/conversationActions';
import { createFriendActions } from './chat/friendActions';
import { createGroupActions } from './chat/groupActions';
import { createProfileActions } from './chat/profileActions';

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
  const [viewingFriendRequests, setViewingFriendRequests] = useState(false);
  const [viewingGroupManage, setViewingGroupManage] = useState(false);
  const [hoverCard, setHoverCard] = useState<{ user: UserDTO; rect: DOMRect } | null>(null);
  const [onlineMap, setOnlineMap] = useState<Record<number, boolean>>({});
  const [allIncomingReqs, setAllIncomingReqs] = useState<FriendRequestDTO[]>([]);
  const [allOutgoingReqs, setAllOutgoingReqs] = useState<FriendRequestDTO[]>([]);

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
    setViewingFriendRequests(false);
    setViewingGroupManage(false);
    setDetailOpen(false);
    setTab('chats');
    setMobilePane('chat');
    markConversationRead(cid);
  }

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

  function sendConversationMessage(conversationID: number, text: string, msgType = MSG_TYPE_TEXT, replyToID = msgType === MSG_TYPE_TEXT ? replyTo?.id ?? 0 : 0) {
    const content = text.trim();
    if (!content) return;
    const clientID = wsRef.current.sendMessage({ conversation_id: conversationID, content, msg_type: msgType, reply_to: replyToID });
    const optimistic: MessageDTO = {
      id: Date.now(), conversation_id: conversationID, seq: Number.MAX_SAFE_INTEGER,
      sender_id: user.id, msg_type: msgType, content, reply_to: replyToID,
      client_id: clientID, created_at: currentSecond(),
    };
    applyIncomingMessage(optimistic);
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
      setAllIncomingReqs(incoming);
      setAllOutgoingReqs(outgoing);
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
      if (beforeSeq === 0) await loadChatMembers(cid, true);
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '消息加载失败' });
    }
  }

  async function loadChatMembers(cid: number, force = false) {
    if (!force && members[cid]) return;
    try { const list = await api.members(cid); setMembers((p) => ({ ...p, [cid]: list })); }
    catch { setMembers((p) => ({ ...p, [cid]: [] })); }
  }

  function handleWsMessage(msg: WsResponse) {
    handleRealtimeMessage(msg, {
      currentUID: user.id, ws: wsRef.current, typingTimers, deletedMessageIDs,
      getMessages: () => messagesRef.current,
      getDisplayName: (uid) => displayName(uid, userCache, friendMap),
      applyIncomingMessage, setLastMessagePreview, setNotice,
      setMessages, setLastMsgMap, setConversations, setTyping, setOnlineMap, refreshBase,
    });
  }

  realtimeHandlerRef.current = handleWsMessage;

  useEffect(() => {
    const off = wsRef.current.on((msg) => realtimeHandlerRef.current(msg));
    wsRef.current.connect(() => { setOnlineMap({}); void refreshBase(); });
    void refreshBase();
    return () => { off(); wsRef.current.close(); };
  }, []);

  useEffect(() => { if (!selectedID) return; void loadMessages(selectedID); void loadChatMembers(selectedID); }, [selectedID]);
  useEffect(() => { if (!selectedID) return; const t = window.setInterval(() => void loadChatMembers(selectedID, true), 5000); return () => window.clearInterval(t); }, [selectedID]);
  useEffect(() => { if ('Notification' in window && Notification.permission === 'default') Notification.requestPermission(); }, []);
  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(null), 3000);
    return () => clearTimeout(t);
  }, [notice]);

  async function searchUsers(kw?: string) {
    const q = (kw ?? searchKeyword).trim();
    if (!q) { setSearchResult([]); return; }
    try { setSearchResult(await api.searchUsers(q)); }
    catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '搜索失败' }); }
  }

  async function startPrivate(uid: number) {
    if (!friendMap[uid]) { setNotice({ kind: 'error', text: '只能给好友发消息，请先添加好友' }); return; }
    try { const cid = await ensurePrivateConversation(uid); openConversation(cid); }
    catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '创建聊天失败' }); }
  }

  async function viewUserProfile(uid: number) {
    try {
      const u = userCache[uid] ?? (await api.getUser(uid));
      if (u) { setUserCache((p) => ({ ...p, [uid]: u })); setViewingUser(u); setSelectedID(null); selectedIDRef.current = null; setMobilePane('chat'); }
    } catch { setNotice({ kind: 'error', text: '获取用户信息失败' }); }
  }

  function viewFriendRequests() {
    setViewingFriendRequests(true);
    setViewingUser(null);
    setViewingGroupManage(false);
    setSelectedID(null);
    selectedIDRef.current = null;
    setMobilePane('chat');
  }

  function viewGroupManage() {
    setViewingGroupManage(true);
    setViewingUser(null);
    setViewingFriendRequests(false);
    setSelectedID(null);
    selectedIDRef.current = null;
    setMobilePane('chat');
  }

  function showHoverCard(uid: number, rect: DOMRect) {
    const u = userCache[uid];
    if (u) setHoverCard({ user: u, rect });
  }

  const deps: ChatStoreDeps = {
    user, onUserChange, selectedID, selectedIDRef, conversations, details,
    userCache, friendMap, friends, friendGroups, members, messages, replyTo,
    chatSearch, deletedMessageIDs, deletedStorageKey, messagesRef, typingTimers, wsRef, realtimeHandlerRef,
    setConversations, setDetails, setUserCache, setSelectedID, setMessages,
    setLastMsgMap, setHasMore, setFriends, setFriendGroups, setRequests,
    setOutgoingReqs, setMembers, setTyping, setDetailOpen, setModal,
    setContextMenu, setReplyTo, setChatSearch, setChatSearchResult, setNotice,
    setViewingUser, setOnlineMap, setTab, setMobilePane,
    refreshBase, loadMessages, loadChatMembers, markConversationRead,
    applyIncomingMessage, setLastMessagePreview, rememberDeletedMessage,
    openConversation, sendConversationMessage, ensurePrivateConversation,
  };

  const msgActions = createMessageActions(deps);
  const convActions = createConversationActions(deps);
  const friendActions = createFriendActions(deps);
  const groupActions = createGroupActions(deps);
  const profileActions = createProfileActions(deps);

  return {
    tab, setTab, mobilePane, setMobilePane,
    conversations, sortedConversations, groupConversations, details, userCache,
    selectedID, selectedConv, messages, lastMsgMap, hasMore, friends, friendGroups, friendMap,
    requests, outgoingReqs, allIncomingReqs, allOutgoingReqs, members, searchKeyword, setSearchKeyword, searchResult,
    typing, loading, detailOpen, setDetailOpen, modal, setModal, contextMenu,
    setContextMenu, replyTo, setReplyTo, chatSearch, setChatSearch, chatSearchResult, setChatSearchResult,
    notice, setNotice, unreadTotal,
    viewingUser, setViewingUser, viewingFriendRequests, setViewingFriendRequests, viewingGroupManage, setViewingGroupManage, hoverCard, setHoverCard, onlineMap,
    refreshBase, loadMessages, loadChatMembers,
    searchUsers, startPrivate, viewUserProfile, viewFriendRequests, viewGroupManage, showHoverCard,
    ...msgActions, ...convActions, ...friendActions, ...groupActions, ...profileActions,
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
