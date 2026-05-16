import { useRef, useState } from 'react';
import { api, getToken } from './api/http';
import { useAuth, useChatStore } from './hooks/useChatStore';
import { chatTitle, displayName } from './utils';
import { AuthPage } from './components/AuthPage';
import { NavRail } from './components/NavRail';
import { ConversationList } from './components/ConversationList';
import { ChatWindow } from './components/ChatWindow';
import { ContactsPanel } from './components/ContactsPanel';
import { ProfilePanel } from './components/ProfilePanel';
import { ChatDetailPanel } from './components/ChatDetailPanel';
import { AppModal, ContextMenuPopup } from './components/Modal';
import { UserCard } from './components/UserCard';
import { UserProfilePage } from './components/UserProfilePage';

function App() {
  const { user, setUser, notice, setNotice, handleLoggedIn, logout } = useAuth();
  if (!user || !getToken()) return <AuthPage onLoggedIn={handleLoggedIn} notice={notice} setNotice={setNotice} />;
  return <ChatPage user={user} onUserChange={setUser} onLogout={logout} />;
}

function ChatPage({ user, onUserChange, onLogout }: { user: import('./api/types').UserDTO; onUserChange: (u: import('./api/types').UserDTO) => void; onLogout: () => void }) {
  const store = useChatStore(user, onUserChange);
  const hoverCloseTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [quickActionOpen, setQuickActionOpen] = useState(false);
  const {
    tab, setTab, mobilePane, setMobilePane,
    sortedConversations, groupConversations, details, userCache,
    selectedID, selectedConv, messages, lastMsgMap, hasMore, friends, friendGroups, friendMap,
    requests, outgoingReqs, members, searchKeyword, setSearchKeyword, searchResult,
    typing, loading, detailOpen, setDetailOpen, modal, setModal, contextMenu,
    setContextMenu, replyTo, setReplyTo, chatSearch, setChatSearch, chatSearchResult, setChatSearchResult,
    showChatSearch, setShowChatSearch, notice, setNotice, unreadTotal,
    viewingUser, setViewingUser, hoverCard, setHoverCard, onlineMap,
    sendText, sendImage, searchUsers, startPrivate, createGroup, addFriendByUsername, handleRequest,
    uploadAvatar, togglePin, toggleMute, markAllRead, renameGroup, inviteMember,
    removeMember, leaveCurrentGroup, dissolveCurrentGroup, setMemberRole, transferOwner,
    uploadGroupAvatar, setMemberLimit, deleteFriend, updateFriendRemark, createFriendGroup,
    renameFriendGroup, deleteFriendGroup, updateProfile, changePassword,
    doRevoke, doDelete, doForward, doChatSearch, selectChat,
    viewUserProfile, showHoverCard,
  } = store;

  const selectedMessages = selectedID ? messages[selectedID] ?? [] : [];
  const selectedDetail = selectedID ? details[selectedID] : undefined;
  const selectedMembers = selectedID ? members[selectedID] ?? [] : [];

  const typingText = selectedID ? typing[selectedID] : '';
  const subtitle = selectedConv
    ? selectedDetail?.type === 1
      ? (() => { const peer = selectedMembers.find((m) => m.uid !== user.id); const base = peer ? `${displayName(peer.uid, userCache, friendMap)} · ${onlineMap[peer.uid] ? '在线' : '离线'}` : '私聊'; return typingText ? typingText : base; })()
      : `${selectedMembers.length} 位成员${typingText ? ' · ' + typingText : ''}`
    : '选择聊天后开始';

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
    <main className={`im-shell ${detailOpen ? 'show-detail' : ''}`} onClick={() => { setContextMenu(null); setHoverCard(null); setQuickActionOpen(false); }}>
      <NavRail user={user} tab={tab} onTab={setTab} onLogout={onLogout} unreadTotal={unreadTotal} onMarkAllRead={markAllRead} />
      <section className={`pane-list ${mobilePane === 'list' || mobilePane === 'contacts' ? 'mobile-show' : ''}`}>
        <div className="list-header">
          <div><strong>{tab === 'chats' ? '消息' : tab === 'contacts' ? '通讯录' : '我'}</strong><span>{loading ? '同步中...' : '在线'}</span></div>
          {tab === 'chats'
            ? <div className="quick-action-wrap" onClick={(e) => e.stopPropagation()}>
              <button className="icon-btn" onClick={() => setQuickActionOpen((open) => !open)}>+</button>
              {quickActionOpen && <div className="quick-action-menu">
                <button onClick={() => { setQuickActionOpen(false); createGroup(); }}>创建群聊</button>
                <button onClick={() => { setQuickActionOpen(false); addFriendByUsername(); }}>添加好友</button>
              </div>}
            </div>
            : <button className="icon-btn" onClick={searchUsers}>搜</button>}
        </div>
        {tab === 'chats' && <ConversationList conversations={sortedConversations} details={details} lastMsgMap={lastMsgMap} selectedID={selectedID} onSelect={selectChat} members={members} userCache={userCache} onlineMap={onlineMap} friendMap={friendMap} currentUID={user.id} onTogglePin={togglePin} onToggleMute={toggleMute} />}
        {tab === 'contacts' && <ContactsPanel currentUID={user.id} keyword={searchKeyword} setKeyword={setSearchKeyword} onSearch={searchUsers} results={searchResult} friends={friends} friendGroups={friendGroups} requests={requests} outgoingReqs={outgoingReqs} groupConversations={groupConversations} details={details} userCache={userCache} friendMap={friendMap} onStartPrivate={startPrivate} onRequest={async (target) => { if (target.id === user.id) return; await api.sendFriendRequestByUsername(target.username, '你好'); setNotice({ kind: 'ok', text: '好友申请已发送' }); }} onHandleRequest={handleRequest} onSelectChat={selectChat} onDeleteFriend={deleteFriend} onUpdateRemark={updateFriendRemark} onCreateGroup={createFriendGroup} onRenameGroup={renameFriendGroup} onDeleteGroup={deleteFriendGroup} onViewUser={viewUserProfile} />}
        {tab === 'profile' && <ProfilePanel user={user} onUploadAvatar={uploadAvatar} onUpdateProfile={updateProfile} onChangePassword={changePassword} />}
      </section>
      <section className={`pane-chat ${mobilePane === 'chat' ? 'mobile-show' : ''}`}>
        {viewingUser
          ? <UserProfilePage user={viewingUser} isFriend={!!friendMap[viewingUser.id]} isSelf={viewingUser.id === user.id} onBack={() => setViewingUser(null)} onStartPrivate={(uid) => { void startPrivate(uid); }} onAddFriend={async () => { if (viewingUser.id === user.id) return; await api.sendFriendRequestByUsername(viewingUser.username, '你好'); setNotice({ kind: 'ok', text: '好友申请已发送' }); setViewingUser(null); }} />
          : <ChatWindow user={user} conversation={selectedConv} detail={selectedDetail} title={selectedConv ? chatTitle(selectedConv, selectedDetail) : '请选择聊天'} subtitle={subtitle} messages={selectedMessages} hasMore={selectedID ? hasMore[selectedID] ?? false : false} typingText={selectedID ? typing[selectedID] : ''} memberCount={selectedMembers.length} members={selectedMembers} userCache={userCache} friendMap={friendMap} detailOpen={detailOpen} replyTo={replyTo} showChatSearch={showChatSearch} chatSearch={chatSearch} chatSearchResult={chatSearchResult} onBack={() => setMobilePane(tab === 'contacts' ? 'contacts' : 'list')} onSend={sendText} onSendImage={sendImage} onTyping={() => selectedID && store.wsRef.current.typing(selectedID)} onToggleDetail={() => setDetailOpen(!detailOpen)} onContextMenu={(e, m) => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, message: m }); }} onReply={setReplyTo} onLoadMore={() => { if (!selectedID) return undefined; const first = messages[selectedID]?.[0]; return first ? store.loadMessages(selectedID, first.seq) : undefined; }} onChatSearch={doChatSearch} onChatSearchChange={setChatSearch} onToggleChatSearch={() => { setShowChatSearch(!showChatSearch); setChatSearchResult([]); }} onJumpToMessage={(id) => { setShowChatSearch(false); setChatSearchResult([]); const el = document.getElementById(`msg-${id}`); if (el) { el.scrollIntoView({ behavior: 'smooth', block: 'center' }); el.classList.add('highlight-msg'); setTimeout(() => el.classList.remove('highlight-msg'), 2000); } }} onAvatarEnter={handleAvatarEnter} onAvatarLeave={scheduleCloseHoverCard} onAvatarClick={(uid) => viewUserProfile(uid)} />
        }
      </section>
      <aside className="pane-detail">
        <ChatDetailPanel chat={selectedConv} detail={selectedDetail} members={selectedMembers} currentUID={user.id} userCache={userCache} friendMap={friendMap} onRenameGroup={renameGroup} onInviteMember={inviteMember} onRemoveMember={removeMember} onLeaveGroup={leaveCurrentGroup} onDissolveGroup={dissolveCurrentGroup} onTogglePin={togglePin} onToggleMute={toggleMute} onSetMemberRole={setMemberRole} onTransferOwner={transferOwner} onUploadGroupAvatar={uploadGroupAvatar} onSetMemberLimit={setMemberLimit} onViewUser={viewUserProfile} onClose={() => setDetailOpen(false)} />
      </aside>
      {notice && <div className={`notice floating ${notice.kind}`} onClick={() => setNotice(null)}>{notice.text}</div>}
      <AppModal modal={modal} onClose={() => setModal(null)} />
      {contextMenu && <ContextMenuPopup menu={contextMenu} mine={contextMenu.message.sender_id === user.id} onRevoke={() => doRevoke(contextMenu.message)} onReply={() => setReplyTo(contextMenu.message)} onCopy={() => navigator.clipboard.writeText(contextMenu.message.content)} onDelete={() => doDelete(contextMenu.message)} onForward={() => doForward(contextMenu.message)} onClose={() => setContextMenu(null)} />}
      {hoverCard && <UserCard data={hoverCard} isFriend={!!friendMap[hoverCard.user.id]} isSelf={hoverCard.user.id === user.id} onMouseEnter={keepHoverCard} onMouseLeave={scheduleCloseHoverCard} onStartPrivate={(uid) => { setHoverCard(null); startPrivate(uid); }} onAddFriend={async () => { if (hoverCard.user.id === user.id) return; await api.sendFriendRequestByUsername(hoverCard.user.username, '你好'); setNotice({ kind: 'ok', text: '好友申请已发送' }); setHoverCard(null); }} onViewProfile={(uid) => { setHoverCard(null); viewUserProfile(uid); }} />}
    </main>
  );
}

export default App;
