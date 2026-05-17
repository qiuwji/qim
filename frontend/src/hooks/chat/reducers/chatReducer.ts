import type { SetStateAction } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import { upsertMessage } from '@/utils';
import { applyIncomingConversation, markConversationReadLocally, visibleConversations } from '../models/conversationModel';
import { applyMessagePreview, applyPreviewTexts, latestVisibleMessage, mergeLoadedMessages, messagePreviewText, removeMessageByID, visibleMessages } from '../models/messageModel';

export interface ChatState {
  conversations: UserConvDTO[];
  details: Record<number, ConversationDTO>;
  userCache: Record<number, UserDTO>;
  selectedID: number | null;
  messages: Record<number, MessageDTO[]>;
  lastMsgMap: Record<number, string>;
  hasMore: Record<number, boolean>;
  friends: FriendDTO[];
  friendGroups: FriendGroupDTO[];
  requests: FriendRequestDTO[];
  outgoingReqs: FriendRequestDTO[];
  allIncomingReqs: FriendRequestDTO[];
  allOutgoingReqs: FriendRequestDTO[];
  members: Record<number, MemberDTO[]>;
  typing: Record<number, string>;
  onlineMap: Record<number, boolean>;
  viewingUser: UserDTO | null;
}

export function createInitialChatState(user: UserDTO): ChatState {
  return {
    conversations: [],
    details: {},
    userCache: { [user.id]: user },
    selectedID: null,
    messages: {},
    lastMsgMap: {},
    hasMore: {},
    friends: [],
    friendGroups: [],
    requests: [],
    outgoingReqs: [],
    allIncomingReqs: [],
    allOutgoingReqs: [],
    members: {},
    typing: {},
    onlineMap: {},
    viewingUser: null,
  };
}

type StateKey = keyof ChatState;

type SetFieldAction<K extends StateKey = StateKey> = {
  type: 'setField';
  key: K;
  value: SetStateAction<ChatState[K]>;
};

export type ChatAction =
  | SetFieldAction
  | { type: 'baseLoaded'; conversations: UserConvDTO[]; friends: FriendDTO[]; incoming: FriendRequestDTO[]; outgoing: FriendRequestDTO[]; friendGroups: FriendGroupDTO[]; hiddenIDs: Set<number> }
  | { type: 'hydrateFriendUsers'; users: Record<number, UserDTO> }
  | { type: 'hydrateChatMeta'; members: Record<number, MemberDTO[]>; details: Record<number, ConversationDTO>; users: Record<number, UserDTO>; previews: Record<number, string | undefined>; chats: UserConvDTO[] }
  | { type: 'hideConversation'; conversationID: number }
  | { type: 'openConversation'; conversationID: number }
  | { type: 'closeConversation' }
  | { type: 'markConversationRead'; conversationID: number }
  | { type: 'incomingMessage'; message: MessageDTO; selectedID: number | null; currentUID: number }
  | { type: 'messagesLoaded'; conversationID: number; rawMessages: MessageDTO[]; beforeSeq: number; pageSize: number; deletedIDs: Set<number> }
  | { type: 'deleteLocalMessage'; message: MessageDTO };

function resolveSetState<T>(current: T, value: SetStateAction<T>): T {
  return typeof value === 'function' ? (value as (prev: T) => T)(current) : value;
}

export function chatReducer(state: ChatState, action: ChatAction): ChatState {
  switch (action.type) {
    case 'setField':
      return { ...state, [action.key]: resolveSetState(state[action.key], action.value) };

    case 'baseLoaded':
      return {
        ...state,
        conversations: visibleConversations(action.conversations, action.hiddenIDs),
        friends: action.friends,
        requests: action.incoming.filter((item) => item.status === 0),
        outgoingReqs: action.outgoing.filter((item) => item.status === 0),
        allIncomingReqs: action.incoming,
        allOutgoingReqs: action.outgoing,
        friendGroups: action.friendGroups,
        onlineMap: ensureOnlineDefaults(action.friends, state.onlineMap),
      };

    case 'hydrateFriendUsers':
      return Object.keys(action.users).length
        ? { ...state, userCache: { ...state.userCache, ...action.users } }
        : state;

    case 'hydrateChatMeta':
      return {
        ...state,
        members: { ...state.members, ...action.members },
        details: { ...state.details, ...action.details },
        userCache: Object.keys(action.users).length ? { ...state.userCache, ...action.users } : state.userCache,
        lastMsgMap: applyPreviewTexts(state.lastMsgMap, action.chats, action.previews),
      };

    case 'hideConversation': {
      const selectedID = state.selectedID === action.conversationID ? null : state.selectedID;
      return {
        ...state,
        conversations: state.conversations.filter((item) => item.conversation_id !== action.conversationID),
        selectedID,
      };
    }

    case 'openConversation':
      return {
        ...state,
        selectedID: action.conversationID,
        viewingUser: null,
        conversations: markConversationReadLocally(state.conversations, action.conversationID),
      };

    case 'closeConversation':
      return { ...state, selectedID: null };

    case 'markConversationRead':
      return { ...state, conversations: markConversationReadLocally(state.conversations, action.conversationID) };

    case 'incomingMessage': {
      const preview = messagePreviewText(action.message);
      return {
        ...state,
        messages: {
          ...state.messages,
          [action.message.conversation_id]: upsertMessage(state.messages[action.message.conversation_id] ?? [], action.message),
        },
        lastMsgMap: preview ? { ...state.lastMsgMap, [action.message.conversation_id]: preview } : state.lastMsgMap,
        typing: action.message.sender_id !== action.currentUID ? { ...state.typing, [action.message.conversation_id]: '' } : state.typing,
        conversations: applyIncomingConversation(state.conversations, action.message, action.selectedID, action.currentUID),
      };
    }

    case 'messagesLoaded': {
      const list = visibleMessages(action.rawMessages, action.deletedIDs);
      const newest = list.length > 0 ? list[0] : undefined;
      const existing = action.beforeSeq > 0 ? (state.messages[action.conversationID] ?? []) : [];
      const messages = {
        ...state.messages,
        [action.conversationID]: mergeLoadedMessages(list, existing, action.beforeSeq),
      };
      const lastMsgMap = action.beforeSeq === 0
        ? applyMessagePreview(state.lastMsgMap, action.conversationID, newest)
        : state.lastMsgMap;
      return {
        ...state,
        messages,
        lastMsgMap,
        hasMore: { ...state.hasMore, [action.conversationID]: action.rawMessages.length >= action.pageSize },
      };
    }

    case 'deleteLocalMessage': {
      const nextList = removeMessageByID(state.messages[action.message.conversation_id] ?? [], action.message.id);
      return {
        ...state,
        messages: { ...state.messages, [action.message.conversation_id]: nextList },
        lastMsgMap: applyMessagePreview(state.lastMsgMap, action.message.conversation_id, latestVisibleMessage(nextList, new Set())),
      };
    }

    default:
      return state;
  }
}

function ensureOnlineDefaults(friends: FriendDTO[], onlineMap: Record<number, boolean>): Record<number, boolean> {
  const next = { ...onlineMap };
  for (const friend of friends) {
    if (next[friend.friend_uid] === undefined) next[friend.friend_uid] = false;
  }
  return next;
}
