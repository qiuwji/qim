import { useCallback } from 'react';
import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import { api } from '@/api/http';
import type { ConversationDTO, FriendDTO, FriendRequestDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import type { Notice } from '@/types';
import { collectMissingFriendUserIDs } from '../models/friendModel';
import { latestVisibleMessage, messagePreviewText, MESSAGE_PAGE_SIZE } from '../models/messageModel';
import type { ChatAction, ChatState } from '../reducers/chatReducer';

async function fetchUsersByID(uids: number[], cached: Record<number, UserDTO>): Promise<Record<number, UserDTO>> {
  const missing = Array.from(new Set(uids)).filter((uid) => uid > 0 && !cached[uid]);
  if (!missing.length) return {};
  const users = await Promise.all(missing.map((uid) => api.getUser(uid).catch(() => null)));
  const next: Record<number, UserDTO> = {};
  for (const userItem of users) if (userItem) next[userItem.id] = userItem;
  return next;
}

export function useChatDataActions({
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
}: {
  user: UserDTO;
  conversations: UserConvDTO[];
  details: Record<number, ConversationDTO>;
  members: Record<number, MemberDTO[]>;
  userCache: Record<number, UserDTO>;
  chatStateRef: MutableRefObject<ChatState>;
  deletedMessageIDs: MutableRefObject<Set<number>>;
  hiddenConversationIDs: MutableRefObject<Set<number>>;
  dispatchChat: Dispatch<ChatAction>;
  setConversations: Dispatch<SetStateAction<UserConvDTO[]>>;
  setDetails: Dispatch<SetStateAction<Record<number, ConversationDTO>>>;
  setUserCache: Dispatch<SetStateAction<Record<number, UserDTO>>>;
  setMembers: Dispatch<SetStateAction<Record<number, MemberDTO[]>>>;
  setLoading: Dispatch<SetStateAction<boolean>>;
  setNotice: Dispatch<SetStateAction<Notice>>;
  requestOnlineFriends: () => void;
  markConversationRead: (cid: number, seq?: number) => void;
}) {
  const hydrateFriendUsers = useCallback(async (
    friendList: FriendDTO[],
    incomingReqs: FriendRequestDTO[],
    outgoingReqs: FriendRequestDTO[],
  ) => {
    const missing = collectMissingFriendUserIDs(friendList, incomingReqs, outgoingReqs, chatStateRef.current.userCache);
    if (!missing.length) return;
    const users = await Promise.all(missing.map((uid) => api.getUser(uid).catch(() => null)));
    const next: Record<number, UserDTO> = {};
    for (const userItem of users) if (userItem) next[userItem.id] = userItem;
    if (Object.keys(next).length) dispatchChat({ type: 'hydrateFriendUsers', users: next });
  }, [chatStateRef, dispatchChat]);

  const hydrateChatMeta = useCallback(async (chatList: UserConvDTO[], friendList?: FriendDTO[]) => {
    const nextMembers: Record<number, MemberDTO[]> = {};
    const nextDetails: Record<number, ConversationDTO> = {};
    const nextUsers: Record<number, UserDTO> = {};
    const nextPreviews: Record<number, string | undefined> = {};
    const snapshot = chatStateRef.current;

    await Promise.all(chatList.map(async (chat) => {
      const id = chat.conversation_id;
      const [memberList, msgList] = await Promise.all([
        api.members(id).catch(() => []),
        api.messages(id, 0, MESSAGE_PAGE_SIZE).catch(() => []),
      ]);
      nextMembers[id] = memberList;
      nextPreviews[id] = messagePreviewText(latestVisibleMessage(msgList, deletedMessageIDs.current));
      Object.assign(nextUsers, await fetchUsersByID([
        ...memberList.map((member) => member.uid),
        ...msgList.map((message) => message.sender_id),
      ], { ...snapshot.userCache, ...nextUsers }));

      const serverDetail = chat.conv ?? snapshot.details[id];
      if (serverDetail?.type === 1) {
        const peer = memberList.find((member) => member.uid !== user.id);
        if (!peer) {
          nextDetails[id] = serverDetail;
          return;
        }

        const peerUser = await api.getUser(peer.uid).catch(() => null) ?? snapshot.userCache[peer.uid];
        if (peerUser) {
          nextUsers[peer.uid] = peerUser;
          const peerName = friendList?.find((friend) => friend.friend_uid === peer.uid)?.remark || peerUser.nickname || peerUser.username;
          nextDetails[id] = { ...serverDetail, id, type: 1, name: peerName, avatar: peerUser.avatar, member_count: memberList.length };
          return;
        }
      }

      nextDetails[id] = serverDetail
        ? { ...serverDetail, member_count: memberList.length }
        : { id, type: 2, name: `群聊 ${id}`, avatar: '', owner_id: 0, member_count: memberList.length, member_limit: 0, max_seq: 0, created_at: chat.last_msg_at };
    }));

    dispatchChat({
      type: 'hydrateChatMeta',
      members: nextMembers,
      details: nextDetails,
      users: nextUsers,
      previews: nextPreviews,
      chats: chatList,
    });
  }, [chatStateRef, deletedMessageIDs, dispatchChat, user.id]);

  const refreshBase = useCallback(async () => {
    setLoading(true);
    try {
      const [convList, friendList, incoming, outgoing, groups] = await Promise.all([
        api.chats(),
        api.friends(),
        api.incomingRequests(),
        api.outgoingRequests(),
        api.friendGroups(),
      ]);
      dispatchChat({
        type: 'baseLoaded',
        conversations: convList,
        friends: friendList,
        incoming,
        outgoing,
        friendGroups: groups,
        hiddenIDs: hiddenConversationIDs.current,
      });
      await hydrateChatMeta(convList, friendList);
      await hydrateFriendUsers(friendList, incoming, outgoing);
      requestOnlineFriends();
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '加载失败' });
    } finally {
      setLoading(false);
    }
  }, [dispatchChat, hiddenConversationIDs, hydrateChatMeta, hydrateFriendUsers, requestOnlineFriends, setLoading, setNotice]);

  const loadChatMembers = useCallback(async (cid: number, force = false) => {
    if (!force && chatStateRef.current.members[cid]) return;
    try {
      const list = await api.members(cid);
      setMembers((prev) => ({ ...prev, [cid]: list }));
    } catch {
      setMembers((prev) => ({ ...prev, [cid]: [] }));
    }
  }, [chatStateRef, setMembers]);

  const loadMessages = useCallback(async (cid: number, beforeSeq = 0) => {
    try {
      const rawList = await api.messages(cid, beforeSeq, MESSAGE_PAGE_SIZE);
      const users = await fetchUsersByID(rawList.map((message) => message.sender_id), chatStateRef.current.userCache);
      if (Object.keys(users).length) dispatchChat({ type: 'hydrateFriendUsers', users });
      dispatchChat({
        type: 'messagesLoaded',
        conversationID: cid,
        rawMessages: rawList,
        beforeSeq,
        pageSize: MESSAGE_PAGE_SIZE,
        deletedIDs: deletedMessageIDs.current,
      });
      const newest = latestVisibleMessage(rawList, deletedMessageIDs.current);
      const maxSeq = newest?.seq ?? 0;
      if (maxSeq > 0 && beforeSeq === 0) markConversationRead(cid, maxSeq);
      if (beforeSeq === 0) await loadChatMembers(cid, true);
    } catch (err) {
      setNotice({ kind: 'error', text: err instanceof Error ? err.message : '消息加载失败' });
    }
  }, [chatStateRef, deletedMessageIDs, dispatchChat, loadChatMembers, markConversationRead, setNotice]);

  const ensurePrivateConversation = useCallback(async (uid: number): Promise<number> => {
    const existing = conversations.find((conv) => {
      if (details[conv.conversation_id]?.type !== 1) return false;
      return (members[conv.conversation_id] ?? []).some((member) => member.uid === uid);
    });
    if (existing) return existing.conversation_id;

    const conv = await api.createPrivateChat(uid);
    const [peer, memberList] = await Promise.all([
      userCache[uid] ? Promise.resolve(userCache[uid]) : api.getUser(uid).catch(() => null),
      api.members(conv.id).catch(() => []),
    ]);
    if (peer) setUserCache((prev) => ({ ...prev, [uid]: peer }));
    setDetails((prev) => ({
      ...prev,
      [conv.id]: peer ? { ...conv, type: 1, name: peer.nickname || peer.username, avatar: peer.avatar } : conv,
    }));
    setMembers((prev) => ({
      ...prev,
      [conv.id]: memberList.length ? memberList : [
        { uid: user.id, role: 1, last_read_seq: 0, join_time: conv.created_at },
        { uid, role: 1, last_read_seq: 0, join_time: conv.created_at },
      ],
    }));
    setConversations((prev) => (
      prev.some((item) => item.conversation_id === conv.id)
        ? prev
        : [{ conversation_id: conv.id, is_pinned: false, is_muted: false, unread_count: 0, last_msg_at: conv.created_at }, ...prev]
    ));
    return conv.id;
  }, [conversations, details, members, setConversations, setDetails, setMembers, setUserCache, user.id, userCache]);

  return {
    refreshBase,
    hydrateChatMeta,
    hydrateFriendUsers,
    loadChatMembers,
    loadMessages,
    ensurePrivateConversation,
  };
}
