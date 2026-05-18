import { useCallback, useMemo, useReducer, useRef, useState } from 'react';
import type { SetStateAction } from 'react';
import { api, clearSession, getSavedUser, saveSession } from '@/api/http';
import { RealtimeClient } from '@/api/ws';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO, WsResponse } from '@/api/types';
import { currentSecond, displayName } from '@/utils';
import type { ContextMenu, MainTab, MobilePane, ModalState, Notice } from '@/types';
import { messageDisplayText } from './chat/models/messageModel';
import {
  forgetHiddenConversationID,
  groupConversations as filterGroupConversations,
  hiddenConversationStorageKey,
  persistHiddenConversationID,
  readHiddenConversationIDs,
  sortConversations,
} from './chat/models/conversationModel';
import { buildFriendMap } from './chat/models/friendModel';
import {
  applyMessagePreview,
  deletedMessageStorageKey,
  MSG_TYPE_TEXT,
  persistDeletedMessageID,
  readDeletedMessageIDs,
} from './chat/models/messageModel';
import { handleRealtimeMessage } from './chat/models/realtimeModel';
import { chatReducer, createInitialChatState, type ChatAction, type ChatState } from './chat/reducers/chatReducer';
import type { ChatStoreDeps } from './chat/types';
import { createMessageActions } from './chat/actions/messageActions';
import { createConversationActions } from './chat/actions/conversationActions';
import { createFriendActions } from './chat/actions/friendActions';
import { createGroupActions } from './chat/actions/groupActions';
import { createProfileActions } from './chat/actions/profileActions';
import { createNavigationActions } from './chat/actions/navigationActions';
import { useChatLifecycleEffects } from './chat/effects/useChatLifecycleEffects';
import { useChatDataActions } from './chat/actions/chatDataActions';

export function useChatStore(user: UserDTO, onUserChange: (u: UserDTO) => void) {
  const wsRef = useRef(new RealtimeClient());
  const selectedIDRef = useRef<number | null>(null);
  const messagesRef = useRef<Record<number, MessageDTO[]>>({});
  const realtimeHandlerRef = useRef<(msg: WsResponse) => void>(() => undefined);
  const typingTimers = useRef<Record<number, ReturnType<typeof setTimeout>>>({});
  const deletedStorageKey = deletedMessageStorageKey(user.id);
  const deletedMessageIDs = useRef<Set<number>>(new Set(readDeletedMessageIDs(deletedStorageKey)));
  const hiddenStorageKey = hiddenConversationStorageKey(user.id);
  const hiddenConversationIDs = useRef<Set<number>>(new Set(readHiddenConversationIDs(hiddenStorageKey)));

  const [chatState, dispatchChat] = useReducer(chatReducer, user, createInitialChatState);
  const chatStateRef = useRef<ChatState>(chatState);
  const {
    conversations, details, userCache, selectedID, messages, lastMsgMap, hasMore,
    friends, friendGroups, requests, outgoingReqs, allIncomingReqs, allOutgoingReqs,
    members, typing, onlineMap, viewingUser, mentionMap,
  } = chatState;

  const setChatField = useCallback(<K extends keyof ChatState>(key: K, value: SetStateAction<ChatState[K]>) => {
    dispatchChat({ type: 'setField', key, value } as ChatAction);
  }, []);

  const setConversations = useCallback((value: SetStateAction<UserConvDTO[]>) => setChatField('conversations', value), [setChatField]);
  const setDetails = useCallback((value: SetStateAction<Record<number, ConversationDTO>>) => setChatField('details', value), [setChatField]);
  const setUserCache = useCallback((value: SetStateAction<Record<number, UserDTO>>) => setChatField('userCache', value), [setChatField]);
  const setSelectedID = useCallback((value: SetStateAction<number | null>) => setChatField('selectedID', value), [setChatField]);
  const setMessages = useCallback((value: SetStateAction<Record<number, MessageDTO[]>>) => setChatField('messages', value), [setChatField]);
  const setLastMsgMap = useCallback((value: SetStateAction<Record<number, string>>) => setChatField('lastMsgMap', value), [setChatField]);
  const setHasMore = useCallback((value: SetStateAction<Record<number, boolean>>) => setChatField('hasMore', value), [setChatField]);
  const setFriends = useCallback((value: SetStateAction<FriendDTO[]>) => setChatField('friends', value), [setChatField]);
  const setFriendGroups = useCallback((value: SetStateAction<FriendGroupDTO[]>) => setChatField('friendGroups', value), [setChatField]);
  const setRequests = useCallback((value: SetStateAction<FriendRequestDTO[]>) => setChatField('requests', value), [setChatField]);
  const setOutgoingReqs = useCallback((value: SetStateAction<FriendRequestDTO[]>) => setChatField('outgoingReqs', value), [setChatField]);
  const setMembers = useCallback((value: SetStateAction<Record<number, MemberDTO[]>>) => setChatField('members', value), [setChatField]);
  const setTyping = useCallback((value: SetStateAction<Record<number, string>>) => setChatField('typing', value), [setChatField]);
  const setViewingUser = useCallback((value: SetStateAction<UserDTO | null>) => setChatField('viewingUser', value), [setChatField]);
  const setOnlineMap = useCallback((value: SetStateAction<Record<number, boolean>>) => setChatField('onlineMap', value), [setChatField]);

  const [tab, setTab] = useState<MainTab>('chats');
  const [mobilePane, setMobilePane] = useState<MobilePane>('list');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResult, setSearchResult] = useState<UserDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [modal, setModal] = useState<ModalState>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenu>(null);
  const [replyTo, setReplyTo] = useState<MessageDTO | null>(null);
  const [chatSearch, setChatSearch] = useState('');
  const [chatSearchResult, setChatSearchResult] = useState<MessageDTO[]>([]);
  const [notice, setNotice] = useState<Notice>(null);
  const [viewingFriendRequests, setViewingFriendRequests] = useState(false);
  const [viewingGroupManage, setViewingGroupManage] = useState(false);
  const [hoverCard, setHoverCard] = useState<{ user: UserDTO; rect: DOMRect } | null>(null);

  const selectedConv = conversations.find((item) => item.conversation_id === selectedID) ?? null;
  const sortedConversations = useMemo(() => sortConversations(conversations), [conversations]);
  const groupConversations = useMemo(() => filterGroupConversations(sortedConversations, details), [sortedConversations, details]);
  const unreadTotal = useMemo(() => conversations.reduce((s, c) => s + c.unread_count, 0), [conversations]);
  const friendMap = useMemo(() => buildFriendMap(friends), [friends]);
  const requestOnlineFriends = useCallback(() => wsRef.current.requestOnlineFriends(), []);

  function rememberDeletedMessage(msg: MessageDTO) {
    persistDeletedMessageID(deletedStorageKey, deletedMessageIDs.current, msg.id);
  }

  function deleteLocalMessage(msg: MessageDTO) {
    dispatchChat({ type: 'deleteLocalMessage', message: msg });
  }

  function hideConversation(cid: number) {
    persistHiddenConversationID(hiddenStorageKey, hiddenConversationIDs.current, cid);
    dispatchChat({ type: 'hideConversation', conversationID: cid });
    if (selectedIDRef.current === cid) {
      selectedIDRef.current = null;
      setDetailOpen(false);
      setMobilePane('list');
    }
  }

  function setLastMessagePreview(cid: number, msg: MessageDTO | undefined) {
    setLastMsgMap((prev) => applyMessagePreview(prev, cid, msg));
  }

  const markConversationRead = useCallback((cid: number, seq = 0) => {
    dispatchChat({ type: 'markConversationRead', conversationID: cid });
    void api.markRead(cid, seq).catch(() => undefined);
  }, []);

  function openConversation(cid: number) {
    dispatchChat({ type: 'openConversation', conversationID: cid });
    selectedIDRef.current = cid;
    setViewingFriendRequests(false);
    setViewingGroupManage(false);
    setDetailOpen(false);
    setTab('chats');
    setMobilePane('chat');
    markConversationRead(cid);
  }

  function setMention(conversationID: number, value: boolean) {
    dispatchChat({ type: 'setMention', conversationID, value });
  }

  function applyIncomingMessage(next: MessageDTO) {
    if (deletedMessageIDs.current.has(next.id)) return;
    const fromSelf = next.sender_id === user.id;
    const wasHidden = hiddenConversationIDs.current.has(next.conversation_id);
    forgetHiddenConversationID(hiddenStorageKey, hiddenConversationIDs.current, next.conversation_id);
    dispatchChat({ type: 'incomingMessage', message: next, selectedID: selectedIDRef.current, currentUID: user.id });
    if (wasHidden) void refreshBase();
    if (selectedIDRef.current === next.conversation_id && next.seq > 0) markConversationRead(next.conversation_id, next.seq);
    if (!fromSelf && selectedIDRef.current !== next.conversation_id && !document.hasFocus()) {
      const conv = conversations.find((c) => c.conversation_id === next.conversation_id);
      const isMuted = conv?.is_muted ?? false;
      if (!isMuted) {
        try { new Notification('QIM 新消息', { body: messageDisplayText(next).slice(0, 50) }); } catch { /* */ }
      }
    }
  }

  function sendConversationMessage(conversationID: number, text: string, msgType = MSG_TYPE_TEXT, replyToID = msgType === MSG_TYPE_TEXT ? replyTo?.id ?? 0 : 0, mentionUIDs?: number[], mentionAll?: boolean) {
    const content = text.trim();
    if (!content) return;
    const clientID = wsRef.current.sendMessage({ conversation_id: conversationID, content, msg_type: msgType, reply_to: replyToID, mention_uids: mentionUIDs, mention_all: mentionAll });
    const optimistic: MessageDTO = {
      id: Date.now(), conversation_id: conversationID, seq: Number.MAX_SAFE_INTEGER,
      sender_id: user.id, msg_type: msgType, content, reply_to: replyToID,
      client_id: clientID, created_at: currentSecond(),
      mention_uids: mentionUIDs, mention_all: mentionAll,
    };
    applyIncomingMessage(optimistic);
  }

  const {
    refreshBase,
    loadChatMembers,
    loadMessages,
    ensurePrivateConversation,
  } = useChatDataActions({
    user,
    conversations,
    details,
    members,
    userCache,
    chatStateRef,
    deletedMessageIDs,
    hiddenConversationIDs,
    dispatchChat,
    setConversations,
    setDetails,
    setUserCache,
    setMembers,
    setLoading,
    setNotice,
    requestOnlineFriends,
    markConversationRead,
  });

  function handleWsMessage(msg: WsResponse) {
    handleRealtimeMessage(msg, {
      currentUID: user.id, ws: wsRef.current, typingTimers, deletedMessageIDs,
      getMessages: () => messagesRef.current,
      getDisplayName: (uid) => displayName(uid, userCache, friendMap),
      applyIncomingMessage, setLastMessagePreview, setNotice,
      setMessages, setLastMsgMap, setConversations, setTyping, setOnlineMap, setMention, refreshBase,
    });
  }

  realtimeHandlerRef.current = handleWsMessage;

  useChatLifecycleEffects({
    chatState, chatStateRef, selectedID, selectedIDRef, messages, messagesRef,
    unreadTotal, wsRef, realtimeHandlerRef, dispatchChat, refreshBase, loadMessages,
    loadChatMembers, notice, setNotice,
  });

  const deps: ChatStoreDeps = {
    user, onUserChange, selectedID, searchKeyword, selectedIDRef, conversations, details,
    userCache, lastMsgMap, friendMap, friends, friendGroups, members, onlineMap, messages, replyTo, mentionMap,
    chatSearch, deletedMessageIDs, deletedStorageKey, messagesRef, typingTimers, wsRef, realtimeHandlerRef,
    setConversations, setDetails, setUserCache, setSelectedID, setMessages,
    setLastMsgMap, setHasMore, setFriends, setFriendGroups, setRequests,
    setOutgoingReqs, setSearchResult, setMembers, setTyping, setDetailOpen, setModal,
    setContextMenu, setReplyTo, setChatSearch, setChatSearchResult, setNotice,
    setViewingUser, setViewingFriendRequests, setViewingGroupManage, setHoverCard, setOnlineMap, setTab, setMobilePane,
    refreshBase, loadMessages, loadChatMembers, markConversationRead,
    applyIncomingMessage, setLastMessagePreview, rememberDeletedMessage, deleteLocalMessage, hideConversation,
    openConversation, sendConversationMessage, ensurePrivateConversation,
  };

  const msgActions = createMessageActions(deps);
  const convActions = createConversationActions(deps);
  const friendActions = createFriendActions(deps);
  const groupActions = createGroupActions(deps);
  const profileActions = createProfileActions(deps);
  const navigationActions = createNavigationActions(deps);

  return {
    tab, setTab, mobilePane, setMobilePane,
    conversations, sortedConversations, groupConversations, details, userCache,
    selectedID, selectedConv, messages, lastMsgMap, hasMore, friends, friendGroups, friendMap,
    requests, outgoingReqs, allIncomingReqs, allOutgoingReqs, members, searchKeyword, setSearchKeyword, searchResult,
    typing, loading, detailOpen, setDetailOpen, modal, setModal, contextMenu,
    setContextMenu, replyTo, setReplyTo, chatSearch, setChatSearch, chatSearchResult, setChatSearchResult,
    notice, setNotice, unreadTotal, mentionMap,
    viewingUser, setViewingUser, viewingFriendRequests, setViewingFriendRequests, viewingGroupManage, setViewingGroupManage, hoverCard, setHoverCard, onlineMap,
    refreshBase, loadMessages, loadChatMembers,
    ...navigationActions,
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
