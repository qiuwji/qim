import { useState } from 'react';
import type { FriendGroupDTO, UserDTO } from '@/api/types';
import { Avatar, PanelHeader } from '@/components/ui';

export function UserProfilePage({ user, isFriend, isSelf, friendGroups, currentGroupId, onBack, onStartPrivate, onAddFriend, onMoveGroup, onDeleteFriend }: {
  user: UserDTO; isFriend?: boolean; isSelf?: boolean; friendGroups?: FriendGroupDTO[]; currentGroupId?: number;
  onBack: () => void; onStartPrivate: (uid: number) => void; onAddFriend?: (uid: number) => void;
  onMoveGroup?: (friendUID: number, groupID: number) => void;
  onDeleteFriend?: (uid: number) => void;
}) {
  const [showGroupPicker, setShowGroupPicker] = useState(false);
  const currentGroup = friendGroups?.find((g) => g.id === currentGroupId);

  return (
    <div className="flex h-full flex-col bg-[#f3f4f6]">
      <PanelHeader title="个人信息" onBack={onBack} />
      <div className="flex-1 overflow-auto">
        <div className="mx-auto flex max-w-lg flex-col items-center gap-5 px-6 py-8">
          <div className="flex w-full flex-col items-center gap-4 rounded-2xl bg-white p-8 shadow-sm">
            <Avatar user={user} large />
            <div className="text-center">
              <div className="text-lg font-semibold text-[#1a1a1a]">{user.nickname || user.username}</div>
              <div className="mt-1 text-sm text-[#999]">账号：{user.username}</div>
            </div>
          </div>

          <div className="w-full rounded-2xl bg-white px-5 py-4 shadow-sm">
            {user.sign
              ? <p className="text-sm leading-relaxed text-[#707987]">{user.sign}</p>
              : <p className="text-sm text-[#b0b5be]">这个人还没有写个性签名</p>
            }
          </div>

          {isFriend && friendGroups && friendGroups.length > 0 && onMoveGroup && (
            <div className="w-full overflow-hidden rounded-2xl bg-white shadow-sm">
              <button
                className="flex w-full items-center justify-between px-5 py-3.5 text-left transition hover:bg-[#f6f8fa]"
                onClick={() => setShowGroupPicker(true)}
              >
                <span className="text-sm text-[#858c98]">好友分组</span>
                <span className="flex items-center gap-1 text-sm text-[#1677c7]">
                  {currentGroup?.name || '默认分组'}
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><polyline points="9 18 15 12 9 6" /></svg>
                </span>
              </button>
            </div>
          )}

          <div className="flex w-full gap-3">
            {!isSelf && isFriend && (
              <button className="flex-1 rounded-xl bg-[#07c160] py-3 text-sm font-semibold text-white hover:bg-[#06ad56] transition" onClick={() => onStartPrivate(user.id)}>发消息</button>
            )}
            {!isSelf && !isFriend && onAddFriend && (
              <button className="flex-1 rounded-xl bg-[#07c160] py-3 text-sm font-semibold text-white hover:bg-[#06ad56] transition" onClick={() => onAddFriend(user.id)}>加好友</button>
            )}
          </div>
          {!isSelf && isFriend && onDeleteFriend && (
            <button className="w-full rounded-xl border border-[#e04344] py-3 text-sm font-semibold text-[#e04344] hover:bg-[#fff1f0] transition" onClick={() => onDeleteFriend(user.id)}>删除好友</button>
          )}
        </div>
      </div>

      {showGroupPicker && isFriend && friendGroups && onMoveGroup && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/35 sm:items-center" onClick={() => setShowGroupPicker(false)}>
          <div className="w-full max-w-sm rounded-t-2xl bg-white shadow-xl sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-[#f0f1f3] px-5 py-3.5">
              <strong className="text-base font-semibold text-[#1a1a1a]">选择分组</strong>
              <button className="grid h-7 w-7 place-items-center rounded-md text-[#b0b5be] hover:bg-[#f0f1f3]" onClick={() => setShowGroupPicker(false)}>
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              </button>
            </div>
            <div className="max-h-64 overflow-auto">
              {friendGroups.map((g) => (
                <button
                  key={g.id}
                  className={`flex w-full items-center justify-between px-5 py-3.5 text-left text-sm transition hover:bg-[#f6f8fa] ${g.id === currentGroupId ? 'text-[#07c160]' : 'text-[#1a1a1a]'}`}
                  onClick={() => { onMoveGroup(user.id, g.id); setShowGroupPicker(false); }}
                >
                  <span>{g.name || '默认分组'}</span>
                  {g.id === currentGroupId && (
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                  )}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}