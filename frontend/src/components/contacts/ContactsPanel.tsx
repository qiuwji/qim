import { useEffect, useMemo, useRef, useState } from 'react';
import type { MouseEvent } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { filterFriendsByKeyword, groupFriendsByGroup, pendingIncomingRequestCount, sortFriendGroups } from '@/hooks/chat/models/contactViewModel';
import { Badge, ContextMenu, SearchBar } from '@/components/ui';
import { ContactSearchResults } from './ContactSearchResults';
import { FriendListSection } from './FriendListSection';
import { GroupListSection } from './GroupListSection';

export function ContactsPanel({ currentUID: _currentUID, keyword, setKeyword, onSearch, results: _results, friends, friendGroups, requests, outgoingReqs: _outgoingReqs, groupConversations, details, userCache, onlineMap, friendMap: _friendMap, mentionMap, onStartPrivate, onRequest: _onRequest, onHandleRequest: _onHandleRequest, onSelectChat, onDeleteFriend, onUpdateRemark, onCreateGroup, onViewUser, onViewFriendRequests, onViewGroupManage }: {
  currentUID: number; keyword: string; setKeyword: (v: string) => void; onSearch: (v: string) => void; results: UserDTO[]; friends: FriendDTO[]; friendGroups: FriendGroupDTO[]; requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>; mentionMap?: Record<number, boolean>;
  onStartPrivate: (uid: number) => void; onRequest: (user: UserDTO) => Promise<void>; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onSelectChat: (id: number) => void;
  onDeleteFriend: (uid: number) => void; onUpdateRemark: (uid: number, cur: string) => void; onCreateGroup: () => void;
  onViewUser?: (uid: number) => void;
  onViewFriendRequests?: () => void;
  onViewGroupManage?: () => void;
}) {
  const [contactsTab, setContactsTab] = useState<'friends' | 'groups'>('friends');
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({});
  const toggle = (k: string) => setOpenSections((p) => ({ ...p, [k]: !p[k] }));
  const [friendCtx, setFriendCtx] = useState<{ x: number; y: number; friend: FriendDTO } | null>(null);
  const [moreOpen, setMoreOpen] = useState(false);
  const noticeCount = pendingIncomingRequestCount(requests);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => onSearch(keyword), 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [keyword, onSearch]);

  const friendsByGroup = useMemo(() => groupFriendsByGroup(friends), [friends]);

  const sortedGroups = useMemo(() => sortFriendGroups(friendGroups), [friendGroups]);

  const filteredFriendsByGroup = useMemo(() => {
    return filterFriendsByKeyword(friendsByGroup, keyword, userCache);
  }, [friendsByGroup, keyword, userCache]);

  const filteredGroupConversations = useMemo(() => {
    if (!keyword) return groupConversations;
    const lower = keyword.toLowerCase();
    return groupConversations.filter((gc) => {
      const title = chatTitle(gc, details[gc.conversation_id]);
      return title.toLowerCase().includes(lower);
    });
  }, [groupConversations, keyword, details]);

  function handleFriendContext(e: MouseEvent, friend: FriendDTO) {
    e.preventDefault();
    setFriendCtx({ x: e.clientX, y: e.clientY, friend });
  }

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#f5f5f5]" onClick={() => { setFriendCtx(null); setMoreOpen(false); }}>
      <div className="shrink-0 border-b border-[#e8e8e8] px-3 py-2.5">
        <SearchBar value={keyword} onChange={setKeyword} placeholder="搜索" />
      </div>

      <button className="flex w-full items-center gap-3 border-b border-[#e8e8e8] px-3 py-2.5 text-left transition hover:bg-[#e8e8e8] mb-2" onClick={() => onViewFriendRequests?.()}>
        <span className="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-[#19c37d] to-[#07a35a] text-base font-bold text-white">新</span>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium text-[#1a1a1a]">新的朋友</div>
          <div className="text-xs text-[#999]">{noticeCount > 0 ? `${noticeCount} 条好友通知` : '好友申请与验证消息'}</div>
        </div>
        <Badge count={noticeCount} />
      </button>

      <div className="flex shrink-0 border-b border-[#e8e8e8] mb-1">
        <button
          className={`flex-1 py-2.5 text-center text-[13px] font-medium transition ${contactsTab === 'friends' ? 'border-b-2 border-[#07c160] text-[#07c160]' : 'text-[#858c98] hover:text-[#1a1a1a]'}`}
          onClick={() => setContactsTab('friends')}
        >好友</button>
        <button
          className={`flex-1 py-2.5 text-center text-[13px] font-medium transition ${contactsTab === 'groups' ? 'border-b-2 border-[#07c160] text-[#07c160]' : 'text-[#858c98] hover:text-[#1a1a1a]'}`}
          onClick={() => setContactsTab('groups')}
        >群聊</button>
      </div>

      <div className="flex-1 overflow-auto">
        {keyword && (
          <ContactSearchResults
            friendsByGroup={filteredFriendsByGroup}
            friendGroups={friendGroups}
            groupConversations={filteredGroupConversations}
            details={details}
            userCache={userCache}
            onlineMap={onlineMap}
            mentionMap={mentionMap}
            onSelectChat={onSelectChat}
            onViewUser={onViewUser}
            onFriendContext={handleFriendContext}
          />
        )}

        {!keyword && contactsTab === 'friends' && (
          <FriendListSection
            friends={friends}
            friendGroups={sortedGroups}
            friendsByGroup={friendsByGroup}
            openSections={openSections}
            moreOpen={moreOpen}
            userCache={userCache}
            onlineMap={onlineMap}
            onToggleSection={toggle}
            onToggleMore={() => setMoreOpen((v) => !v)}
            onCreateGroup={() => { setMoreOpen(false); onCreateGroup(); }}
            onViewGroupManage={() => { setMoreOpen(false); onViewGroupManage?.(); }}
            onViewUser={onViewUser}
            onFriendContext={handleFriendContext}
          />
        )}

        {!keyword && contactsTab === 'groups' && (
          <GroupListSection
            conversations={filteredGroupConversations}
            details={details}
            mentionMap={mentionMap}
            onSelectChat={onSelectChat}
          />
        )}
      </div>

      {friendCtx && <ContextMenu x={friendCtx.x} y={friendCtx.y} onClose={() => setFriendCtx(null)} items={[
        { label: '发消息', action: () => onStartPrivate(friendCtx.friend.friend_uid) },
        { label: '修改备注', action: () => onUpdateRemark(friendCtx.friend.friend_uid, friendCtx.friend.remark) },
        { label: '删除好友', action: () => onDeleteFriend(friendCtx.friend.friend_uid), danger: true },
      ]} />}

    </div>
  );
}
