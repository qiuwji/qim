import { useRef, useState } from 'react';
import { getToken } from '@/api/http';
import { scrollToMessage } from '@/utils';
import { useAuth, useChatStore } from '@/hooks/useChatStore';
import { useCallStore } from '@/hooks/useCallStore';
import { displayName } from '@/utils';
import { AuthPage } from '@/components/auth/AuthPage';
import { NavRail } from '@/components/NavRail';
import { ConversationList } from '@/components/conversation/ConversationList';
import { ContactsPanel } from '@/components/contacts/ContactsPanel';
import { ProfilePanel } from '@/components/user/ProfilePanel';
import { ChatDetailPanel } from '@/components/chat/ChatDetailPanel';
import { AppModal, ContextMenuPopup } from '@/components/ui/Modal';
import { UserCard } from '@/components/user/UserCard';
import { RightPane, type RightPaneView } from '@/components/RightPane';
import { IncomingCallModal } from '@/components/call/IncomingCallModal';
import { CallingView } from '@/components/call/CallingView';
import { CallView } from '@/components/call/CallView';
import { callTypeLabel } from '@/hooks/chat/models/callModel';

function App() {
  const { user, setUser, notice, setNotice, handleLoggedIn, logout } = useAuth();
  if (!user || !getToken()) return <AuthPage onLoggedIn={handleLoggedIn} notice={notice} setNotice={setNotice} />;
  return <ChatPage user={user} onUserChange={setUser} onLogout={logout} />;
}

function ChatPage({ user, onUserChange, onLogout }: { user: import('@/api/types').UserDTO; onUserChange: (u: import('@/api/types').UserDTO) => void; onLogout: () => void }) {
  const store = useChatStore(user, onUserChange);
  const callStore = useCallStore(store.wsRef);
  store.callHandlerRef.current = callStore.handleCallWs;

  const hoverCloseTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [quickActionOpen, setQuickActionOpen] = useState(false);
  const {
    tab, setTab, mobilePane, setMobilePane,
    sortedConversations, groupConversations, details, userCache,
    selectedID, selectedConv, messages, lastMsgMap, hasMore, friends, friendGroups, friendMap,
    requests, outgoingReqs, members, searchKeyword, setSearchKeyword, searchResult,
    typing, loading, detailOpen, setDetailOpen, modal, setModal, contextMenu,
    setContextMenu, replyTo, setReplyTo, chatSearch, setChatSearch, chatSearchResult, setChatSearchResult,
    notice, setNotice, unreadTotal,
    viewingUser, setViewingUser, viewingFriendRequests, setViewingFriendRequests, viewingGroupManage, setViewingGroupManage, allIncomingReqs, allOutgoingReqs, viewFriendRequests, viewGroupManage, hoverCard, setHoverCard, onlineMap,
    sendText, sendImage, searchUsers, startPrivate, createGroup, addFriendByUsername, addFriendByUser, handleRequest,
    uploadAvatar, togglePin, toggleMute, hideChat, markAllRead, renameGroup, inviteMember,
    removeMember, leaveCurrentGroup, dissolveCurrentGroup, setMemberRole, transferOwner,
    uploadGroupAvatar, setMemberLimit, deleteFriend, updateFriendRemark, createFriendGroup,
    renameFriendGroup, deleteFriendGroup, updateProfile, changePassword,
    doRevoke, doDelete, doForward, doChatSearch, selectChat,
    viewUserProfile, showHoverCard, moveFriendGroup, loadMessages,
    mentionMap,
  } = store;

  const selectedMessages = selectedID ? messages[selectedID] ?? [] : [];
  const selectedDetail = selectedID ? details[selectedID] : undefined;
  const selectedMembers = selectedID ? members[selectedID] ?? [] : [];

  const rightPaneView: RightPaneView = viewingGroupManage ? 'group-manage'
    : viewingFriendRequests ? 'friend-requests'
    : viewingUser ? 'user-profile'
    : 'chat';

  function handleAvatarEnter(uid: number, e: React.MouseEvent) {
    if (hoverCloseTimer.current) {
      clearTimeout(hoverCloseTimer.current);
      hoverCloseTimer.current = null;
    }
    const el = e.currentTarget as HTMLElement;
    if (el) showHoverCard(uid, el.getBoundingClientRect());
  }

  function keepHoverCard() {
    if (hoverCloseTimer.current) {
      clearTimeout(hoverCloseTimer.current);
      hoverCloseTimer.current = null;
    }
  }

  function scheduleCloseHoverCard() {
    if (hoverCloseTimer.current) clearTimeout(hoverCloseTimer.current);
    hoverCloseTimer.current = setTimeout(() => setHoverCard(null), 180);
  }

  return (
    <main className={`im-shell ${detailOpen ? 'show-detail' : ''} ${mobilePane === 'chat' ? 'nav-hidden' : ''}`} onClick={() => { setContextMenu(null); setHoverCard(null); setQuickActionOpen(false); setDetailOpen(false); }}>
      <NavRail user={user} tab={tab} onTab={(t) => { setTab(t); if (t === 'profile') { setViewingUser(null); setViewingFriendRequests(false); setViewingGroupManage(false); } }} onLogout={onLogout} unreadTotal={unreadTotal} onMarkAllRead={markAllRead} className={mobilePane === 'chat' ? 'nav-hidden' : ''} />
      <section className={`pane-list ${mobilePane === 'list' || mobilePane === 'contacts' ? 'mobile-show' : ''}`}>
        <div className="list-header">
          <div><strong>{tab === 'chats' ? '消息' : tab === 'contacts' ? '通讯录' : '我'}</strong><span>{loading ? '同步中...' : '在线'}</span></div>
          {tab === 'chats' && <div className="quick-action-wrap" onClick={(e) => e.stopPropagation()}>
              <button className="icon-btn" onClick={() => setQuickActionOpen((open) => !open)}>+</button>
              {quickActionOpen && <div className="quick-action-menu">
                <button onClick={() => { setQuickActionOpen(false); createGroup(); }}>创建群聊</button>
                <button onClick={() => { setQuickActionOpen(false); addFriendByUsername(); }}>添加好友</button>
              </div>}
            </div>}
        </div>
        {tab === 'chats' && <ConversationList conversations={sortedConversations} details={details} lastMsgMap={lastMsgMap} selectedID={selectedID} onSelect={selectChat} members={members} userCache={userCache} onlineMap={onlineMap} friendMap={friendMap} currentUID={user.id} mentionMap={mentionMap} onTogglePin={togglePin} onToggleMute={toggleMute} onHide={hideChat} />}
        {tab === 'contacts' && <ContactsPanel currentUID={user.id} keyword={searchKeyword} setKeyword={setSearchKeyword} onSearch={searchUsers} results={searchResult} friends={friends} friendGroups={friendGroups} requests={requests} outgoingReqs={outgoingReqs} groupConversations={groupConversations} details={details} userCache={userCache} onlineMap={onlineMap} friendMap={friendMap} mentionMap={mentionMap} onStartPrivate={startPrivate} onRequest={addFriendByUser} onHandleRequest={handleRequest} onSelectChat={selectChat} onDeleteFriend={deleteFriend} onUpdateRemark={updateFriendRemark} onCreateGroup={createFriendGroup} onViewUser={viewUserProfile} onViewFriendRequests={() => viewingFriendRequests ? (setViewingFriendRequests(false), setMobilePane('contacts')) : viewFriendRequests()} onViewGroupManage={viewGroupManage} />}
        {tab === 'profile' && <ProfilePanel user={user} onUploadAvatar={uploadAvatar} onUpdateProfile={updateProfile} onChangePassword={changePassword} />}
      </section>
      <section className={`pane-chat ${mobilePane === 'chat' ? 'mobile-show' : ''}`}>
        <RightPane
          view={rightPaneView}
          user={user}
          tab={tab}
          setMobilePane={setMobilePane}
          selectedConv={selectedConv}
          selectedDetail={selectedDetail}
          selectedMessages={selectedMessages}
          selectedMembers={selectedMembers}
          selectedID={selectedID}
          hasMore={selectedID ? hasMore[selectedID] ?? false : false}
          typingText={selectedID ? typing[selectedID] : ''}
          members={selectedMembers}
          userCache={userCache}
          friendMap={friendMap}
          onlineMap={onlineMap}
          detailOpen={detailOpen}
          replyTo={replyTo}
          viewingUser={viewingUser}
          setViewingUser={setViewingUser}
          setViewingFriendRequests={setViewingFriendRequests}
          setViewingGroupManage={setViewingGroupManage}
          allIncomingReqs={allIncomingReqs}
          allOutgoingReqs={allOutgoingReqs}
          friendGroups={friendGroups}
          sendText={sendText}
          sendImage={sendImage}
          handleRequest={handleRequest}
          viewUserProfile={viewUserProfile}
          startPrivate={startPrivate}
          addFriendByUser={addFriendByUser}
          deleteFriend={deleteFriend}
          moveFriendGroup={moveFriendGroup}
          renameFriendGroup={renameFriendGroup}
          deleteFriendGroup={deleteFriendGroup}
          setDetailOpen={setDetailOpen}
          setReplyTo={setReplyTo}
          setContextMenu={setContextMenu}
          loadMessages={loadMessages}
          wsRef={store.wsRef}
          onAvatarEnter={handleAvatarEnter}
          onAvatarLeave={scheduleCloseHoverCard}
          callState={callStore.callState}
          onVoiceCall={() => {
            const peer = selectedMembers.find(m => m.uid !== user.id);
            if (peer) callStore.initiateCall(peer.uid, 1);
          }}
          onVideoCall={() => {
            const peer = selectedMembers.find(m => m.uid !== user.id);
            if (peer) callStore.initiateCall(peer.uid, 2);
          }}
        />
      </section>
      <aside className="pane-detail" onClick={(e) => e.stopPropagation()}>
        <ChatDetailPanel chat={selectedConv} detail={selectedDetail} members={selectedMembers} currentUID={user.id} userCache={userCache} friendMap={friendMap} chatSearch={chatSearch} chatSearchResult={chatSearchResult} onChatSearch={doChatSearch} onChatSearchChange={setChatSearch} onJumpToMessage={(id) => { setDetailOpen(false); setChatSearchResult([]); scrollToMessage(id); }} onRenameGroup={renameGroup} onInviteMember={inviteMember} onRemoveMember={removeMember} onLeaveGroup={leaveCurrentGroup} onDissolveGroup={dissolveCurrentGroup} onTogglePin={togglePin} onToggleMute={toggleMute} onHide={hideChat} onSetMemberRole={setMemberRole} onTransferOwner={transferOwner} onUploadGroupAvatar={uploadGroupAvatar} onSetMemberLimit={setMemberLimit} onViewUser={viewUserProfile} onClose={() => setDetailOpen(false)} />
      </aside>
      {notice && <div className={`notice floating ${notice.kind}`} onClick={() => setNotice(null)}>{notice.text}</div>}
      <AppModal modal={modal} onClose={() => setModal(null)} />
      {contextMenu && <ContextMenuPopup menu={contextMenu} mine={contextMenu.message.sender_id === user.id} onRevoke={() => doRevoke(contextMenu.message)} onReply={() => setReplyTo(contextMenu.message)} onCopy={() => navigator.clipboard.writeText(contextMenu.message.content)} onDelete={() => doDelete(contextMenu.message)} onForward={() => doForward(contextMenu.message)} onClose={() => setContextMenu(null)} />}
      {hoverCard && <UserCard data={hoverCard} isFriend={!!friendMap[hoverCard.user.id]} isSelf={hoverCard.user.id === user.id} onMouseEnter={keepHoverCard} onMouseLeave={scheduleCloseHoverCard} onStartPrivate={(uid) => { setHoverCard(null); startPrivate(uid); }} onAddFriend={() => { addFriendByUser(hoverCard.user); setHoverCard(null); }} onViewProfile={(uid) => { setHoverCard(null); viewUserProfile(uid); }} />}
      {callStore.callState === 'incoming' && callStore.activeCall && (
        <IncomingCallModal
          call={callStore.activeCall}
          peerName={displayName(callStore.activeCall.peer_uid, userCache, friendMap)}
          peerAvatar={userCache[callStore.activeCall.peer_uid]?.avatar || ''}
          onAccept={callStore.acceptCall}
          onReject={callStore.rejectCall}
        />
      )}
      {callStore.callState === 'calling' && callStore.activeCall && (
        <CallingView
          peerName={displayName(callStore.activeCall.peer_uid, userCache, friendMap)}
          peerAvatar={userCache[callStore.activeCall.peer_uid]?.avatar || ''}
          callTypeLabel={callTypeLabel(callStore.activeCall.call_type)}
          onCancel={callStore.cancelCall}
        />
      )}
      {callStore.callState === 'connected' && callStore.activeCall && (
        <CallView call={callStore.activeCall} remoteStream={callStore.remoteStream} localStreamRef={callStore.localStreamRef} isMuted={callStore.isMuted} isCameraOff={callStore.isCameraOff} duration={callStore.callDuration} peerName={displayName(callStore.activeCall.peer_uid, userCache, friendMap)} peerAvatar={userCache[callStore.activeCall.peer_uid]?.avatar || ''} facingMode={callStore.facingMode} onFlipCamera={callStore.flipCamera} onToggleMute={callStore.toggleMute} onToggleCamera={callStore.toggleCamera} onHangup={callStore.endCall} />
      )}
    </main>
  );
}

export default App;
