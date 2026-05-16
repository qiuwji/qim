import { useMemo, useState } from 'react';
import { ReactNode } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, UserConvDTO, UserDTO } from '../api/types';
import { chatTitle, shortName } from '../utils';
import { Avatar } from './Avatar';
import { CollapsibleSection } from './CollapsibleSection';

export function ContactsPanel({ currentUID, keyword, setKeyword, onSearch, results, friends, friendGroups, requests, outgoingReqs, groupConversations, details, userCache, friendMap, onStartPrivate, onRequest, onHandleRequest, onSelectChat, onDeleteFriend, onUpdateRemark, onCreateGroup, onRenameGroup, onDeleteGroup, onViewUser }: {
  currentUID: number; keyword: string; setKeyword: (v: string) => void; onSearch: () => void; results: UserDTO[]; friends: FriendDTO[]; friendGroups: FriendGroupDTO[]; requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>;
  onStartPrivate: (uid: number) => void; onRequest: (uid: number) => Promise<void>; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onSelectChat: (id: number) => void;
  onDeleteFriend: (uid: number) => void; onUpdateRemark: (uid: number, cur: string) => void; onCreateGroup: () => void; onRenameGroup: (id: number, cur: string) => void; onDeleteGroup: (id: number) => void;
  onViewUser?: (uid: number) => void;
}) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({ search: true, requests: true, outgoing: true, groups: true, friends: true, fg_0: true });
  const toggle = (k: string) => setOpenSections((p) => ({ ...p, [k]: !p[k] }));

  const friendsByGroup = useMemo(() => {
    const map: Record<number, FriendDTO[]> = { 0: [] };
    for (const f of friends) { const gid = f.group_id || 0; if (!map[gid]) map[gid] = []; map[gid].push(f); }
    return map;
  }, [friends]);

  return (
    <div className="contacts-panel">
      <div className="search-box"><input value={keyword} onChange={(e) => setKeyword(e.target.value)} placeholder="搜索用户名或昵称" onKeyDown={(e) => e.key === 'Enter' && onSearch()} /><button onClick={onSearch}>搜索</button></div>
      {results.length > 0 && <CollapsibleSection title="搜索结果" count={results.length} open={openSections.search ?? true} onToggle={() => toggle('search')}>{results.map((item) => { const isFriend = !!friendMap?.[item.id]; const isSelf = item.id === currentUID; return (<UserLine key={item.id} user={item} actions={<>{!isSelf && isFriend && <button onClick={() => onStartPrivate(item.id)}>聊天</button>}{!isSelf && !isFriend && <button onClick={() => void onRequest(item.id)}>加好友</button>}</>} onViewUser={onViewUser} />); })}</CollapsibleSection>}
      {requests.length > 0 && <CollapsibleSection title="新的朋友" count={requests.length} open={openSections.requests ?? true} onToggle={() => toggle('requests')}>{requests.map((item) => { const fu = userCache[item.from_uid]; return (<div className="contact-item" key={item.id}><div className="contact-info clickable" onClick={() => onViewUser?.(item.from_uid)}><div className="avatar fallback small">{shortName(fu?.nickname || fu?.username)}</div><div className="contact-text"><strong>{fu?.nickname || `用户 ${item.from_uid}`}</strong><span>{item.message || '请求添加你为好友'}</span></div></div><div className="line-actions"><button onClick={() => onHandleRequest(item.id, 'accept')}>同意</button><button onClick={() => onHandleRequest(item.id, 'reject')}>拒绝</button></div></div>); })}</CollapsibleSection>}
      {outgoingReqs.length > 0 && <CollapsibleSection title="已发申请" count={outgoingReqs.length} open={openSections.outgoing ?? true} onToggle={() => toggle('outgoing')}>{outgoingReqs.map((item) => { const tu = userCache[item.to_uid]; return (<div className="contact-item" key={item.id}><div className="contact-info clickable" onClick={() => onViewUser?.(item.to_uid)}><div className="avatar fallback small">{shortName(tu?.nickname || tu?.username)}</div><div className="contact-text"><strong>{tu?.nickname || `用户 ${item.to_uid}`}</strong><span>等待对方同意</span></div></div></div>); })}</CollapsibleSection>}
      {groupConversations.length > 0 && <CollapsibleSection title="我的群聊" count={groupConversations.length} open={openSections.groups ?? true} onToggle={() => toggle('groups')}>{groupConversations.map((item) => (<button key={item.conversation_id} className="contact-item" onClick={() => onSelectChat(item.conversation_id)}><div className="avatar fallback small group-icon">群</div><div className="contact-text"><strong>{chatTitle(item, details[item.conversation_id])}</strong></div></button>))}</CollapsibleSection>}
      <CollapsibleSection title="好友" count={friends.length} open={openSections.friends ?? true} onToggle={() => toggle('friends')} extraActions={<button className="tiny-btn" onClick={onCreateGroup}>新建分组</button>}>
        {friendGroups.map((g) => (<CollapsibleSection key={g.id} title={g.name || '默认分组'} count={(friendsByGroup[g.id] ?? []).length} open={openSections[`fg_${g.id}`] ?? true} onToggle={() => toggle(`fg_${g.id}`)} extraActions={<>{g.id > 0 && <><button className="tiny-btn" onClick={() => onRenameGroup(g.id, g.name)}>改名</button><button className="tiny-btn danger-text" onClick={() => onDeleteGroup(g.id)}>删除</button></>}</>}>
          {(friendsByGroup[g.id] ?? []).map((item) => { const fu = userCache[item.friend_uid]; const dn = item.remark || fu?.nickname || fu?.username || `用户 ${item.friend_uid}`; return (<div className="contact-item" key={item.id}><div className="contact-info clickable" onClick={() => onViewUser?.(item.friend_uid)}><div className="avatar fallback small">{shortName(dn)}</div><div className="contact-text"><strong>{dn}</strong>{fu && <span>@{fu.username}</span>}</div></div><div className="line-actions"><button onClick={() => onStartPrivate(item.friend_uid)}>发消息</button><button onClick={() => onUpdateRemark(item.friend_uid, item.remark)}>备注</button><button className="danger-text" onClick={() => onDeleteFriend(item.friend_uid)}>删除</button></div></div>); })}
        </CollapsibleSection>))}
        {(friendsByGroup[0] ?? []).map((item) => { const fu = userCache[item.friend_uid]; const dn = item.remark || fu?.nickname || fu?.username || `用户 ${item.friend_uid}`; return (<div className="contact-item" key={item.id}><div className="contact-info clickable" onClick={() => onViewUser?.(item.friend_uid)}><div className="avatar fallback small">{shortName(dn)}</div><div className="contact-text"><strong>{dn}</strong>{fu && <span>@{fu.username}</span>}</div></div><div className="line-actions"><button onClick={() => onStartPrivate(item.friend_uid)}>发消息</button><button onClick={() => onUpdateRemark(item.friend_uid, item.remark)}>备注</button><button className="danger-text" onClick={() => onDeleteFriend(item.friend_uid)}>删除</button></div></div>); })}
        {friends.length === 0 && <div className="empty-hint">暂无好友，搜索用户名添加</div>}
      </CollapsibleSection>
    </div>
  );
}

function UserLine({ user, actions, onViewUser }: { user: UserDTO; actions: ReactNode; onViewUser?: (uid: number) => void }) {
  return (<div className="user-line"><Avatar user={user} className="avatar-interactive" onClick={() => onViewUser?.(user.id)} /><div className="clickable" onClick={() => onViewUser?.(user.id)}><strong>{user.nickname || user.username}</strong><span>@{user.username}</span></div><div className="line-actions">{actions}</div></div>);
}
