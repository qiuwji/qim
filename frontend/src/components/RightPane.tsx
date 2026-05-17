import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import type { MobilePane } from '@/types';
import { chatTitle } from '@/utils';
import { conversationSubtitle } from '@/hooks/chat/models/conversationViewModel';
import { ChatWindow } from '@/components/ChatWindow';
import { UserProfilePage } from '@/components/UserProfilePage';
import { FriendRequestsView } from '@/components/FriendRequestsView';
import { GroupManagePanel } from '@/components/GroupManagePanel';

export type RightPaneView = 'chat' | 'friend-requests' | 'group-manage' | 'user-profile';

interface RightPaneProps {
  view: RightPaneView;
  user: UserDTO;
  tab: string;
  setMobilePane: (v: MobilePane) => void;
  selectedConv: UserConvDTO | null;
  selectedDetail: ConversationDTO | undefined;
  selectedMessages: MessageDTO[];
  selectedMembers: MemberDTO[];
  selectedID: number | null;
  hasMore: boolean;
  typingText: string;
  userCache: Record<number, UserDTO>;
  friendMap: Record<number, FriendDTO>;
  onlineMap: Record<number, boolean>;
  detailOpen: boolean;
  replyTo: MessageDTO | null;
  viewingUser: UserDTO | null;
  setViewingUser: (u: UserDTO | null) => void;
  setViewingFriendRequests: (v: boolean) => void;
  setViewingGroupManage: (v: boolean) => void;
  allIncomingReqs: FriendRequestDTO[];
  allOutgoingReqs: FriendRequestDTO[];
  friendGroups: FriendGroupDTO[];
  sendText: (text: string) => void;
  sendImage: (file: File) => void;
  handleRequest: (id: number, a: 'accept' | 'reject') => void;
  viewUserProfile: (uid: number) => void;
  startPrivate: (uid: number) => void;
  addFriendByUser: (u: UserDTO) => void;
  deleteFriend: (uid: number) => void;
  moveFriendGroup: (uid: number, gid: number) => void;
  renameFriendGroup: (id: number, cur: string) => void;
  deleteFriendGroup: (id: number) => void;
  setDetailOpen: (v: boolean) => void;
  setReplyTo: (m: MessageDTO | null) => void;
  setContextMenu: (v: { x: number; y: number; message: MessageDTO } | null) => void;
  loadMessages: (cid: number, before: number) => Promise<void>;
  wsRef: { current: { typing: (cid: number) => void } };
  onAvatarEnter: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave: () => void;
}

export function RightPane(props: RightPaneProps) {
  const {
    view, user, tab, setMobilePane,
    selectedConv, selectedDetail, selectedMessages, selectedMembers, selectedID, hasMore, typingText,
    userCache, friendMap, onlineMap, detailOpen, replyTo,
    viewingUser, setViewingUser, setViewingFriendRequests, setViewingGroupManage,
    allIncomingReqs, allOutgoingReqs, friendGroups,
    sendText, sendImage, handleRequest, viewUserProfile, startPrivate, addFriendByUser, deleteFriend,
    moveFriendGroup, renameFriendGroup, deleteFriendGroup,
    setDetailOpen, setReplyTo, setContextMenu, loadMessages, wsRef,
    onAvatarEnter, onAvatarLeave,
  } = props;

  const subtitle = selectedConv ? conversationSubtitle({
    detail: selectedDetail,
    members: selectedMembers,
    currentUID: user.id,
    userCache,
    friendMap,
    onlineMap,
    typingText,
  }) : '';

  const title = selectedConv ? chatTitle(selectedConv, selectedDetail) : '';

  function goBack() {
    setMobilePane(tab === 'contacts' ? 'contacts' : 'list');
  }

  if (view === 'group-manage') {
    return (
      <GroupManagePanel
        groups={friendGroups}
        onBack={() => { setViewingGroupManage(false); setMobilePane('contacts'); }}
        onRenameGroup={renameFriendGroup}
        onDeleteGroup={deleteFriendGroup}
      />
    );
  }

  if (view === 'friend-requests') {
    return (
      <FriendRequestsView
        currentUID={user.id}
        incoming={allIncomingReqs}
        outgoing={allOutgoingReqs}
        userCache={userCache}
        onBack={() => { setViewingFriendRequests(false); setMobilePane('contacts'); }}
        onHandleRequest={handleRequest}
        onViewUser={viewUserProfile}
      />
    );
  }

  if (view === 'user-profile' && viewingUser) {
    return (
      <UserProfilePage
        user={viewingUser}
        isFriend={!!friendMap[viewingUser.id]}
        isSelf={viewingUser.id === user.id}
        friendGroups={friendGroups}
        currentGroupId={friendMap[viewingUser.id]?.group_id}
        onBack={() => { setViewingUser(null); goBack(); }}
        onStartPrivate={(uid) => { void startPrivate(uid); }}
        onAddFriend={() => { addFriendByUser(viewingUser); setViewingUser(null); }}
        onMoveGroup={moveFriendGroup}
        onDeleteFriend={deleteFriend}
      />
    );
  }

  return (
    <ChatWindow
      user={user}
      conversation={selectedConv}
      detail={selectedDetail}
      title={title}
      subtitle={subtitle}
      messages={selectedMessages}
      hasMore={selectedID ? hasMore : false}
      typingText={selectedID ? typingText : ''}
      userCache={userCache}
      friendMap={friendMap}
      detailOpen={detailOpen}
      replyTo={replyTo}
      onBack={goBack}
      onSend={(text) => { sendText(text); return Promise.resolve(); }}
      onSendImage={sendImage}
      onTyping={() => selectedID && wsRef.current.typing(selectedID)}
      onToggleDetail={() => setDetailOpen(!detailOpen)}
      onContextMenu={(e, m) => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, message: m }); }}
      onReply={setReplyTo}
      onLoadMore={() => {
        if (!selectedID) return undefined;
        const first = selectedMessages[0];
        return first ? loadMessages(selectedID, first.seq) : undefined;
      }}
      onAvatarEnter={onAvatarEnter}
      onAvatarLeave={onAvatarLeave}
      onAvatarClick={(uid) => viewUserProfile(uid)}
    />
  );
}
