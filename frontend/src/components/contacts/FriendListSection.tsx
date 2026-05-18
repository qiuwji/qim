import React from 'react';
import type { FriendDTO, FriendGroupDTO, UserDTO } from '@/api/types';
import { CollapsibleSection, EmptyState } from '@/components/ui';
import { ContactFriendItem } from './ContactFriendItem';

export function FriendListSection({
  friends,
  friendGroups,
  friendsByGroup,
  openSections,
  moreOpen,
  userCache,
  onlineMap,
  onToggleSection,
  onToggleMore,
  onCreateGroup,
  onViewGroupManage,
  onViewUser,
  onFriendContext,
}: {
  friends: FriendDTO[];
  friendGroups: FriendGroupDTO[];
  friendsByGroup: Record<number, FriendDTO[]>;
  openSections: Record<string, boolean>;
  moreOpen: boolean;
  userCache: Record<number, UserDTO>;
  onlineMap: Record<number, boolean>;
  onToggleSection: (key: string) => void;
  onToggleMore: () => void;
  onCreateGroup: () => void;
  onViewGroupManage?: () => void;
  onViewUser?: (uid: number) => void;
  onFriendContext: (e: React.MouseEvent, friend: FriendDTO) => void;
}) {
  return (
    <>
      <div className="flex items-center justify-between px-3 py-1.5">
        <span className="text-xs font-semibold text-[#858c98]">好友分组</span>
        <div className="relative">
          <button className="grid h-6 w-6 place-items-center rounded text-[#858c98] hover:bg-[#e0e0e0]" onClick={(e) => { e.stopPropagation(); onToggleMore(); }}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><circle cx="12" cy="5" r="1.5" /><circle cx="12" cy="12" r="1.5" /><circle cx="12" cy="19" r="1.5" /></svg>
          </button>
          {moreOpen && (
            <div className="absolute right-0 top-full z-20 mt-1 min-w-[120px] rounded-lg border border-[#e8e8e8] bg-white py-1 shadow-lg" onClick={(e) => e.stopPropagation()}>
              <button className="w-full px-4 py-2 text-left text-xs text-[#1a1a1a] hover:bg-[#f5f5f5]" onClick={onCreateGroup}>新建分组</button>
              <button className="w-full px-4 py-2 text-left text-xs text-[#1a1a1a] hover:bg-[#f5f5f5]" onClick={onViewGroupManage}>管理分组</button>
            </div>
          )}
        </div>
      </div>

      {friendGroups.map((group) => (
        <CollapsibleSection key={group.id} title={group.name || '默认分组'} count={(friendsByGroup[group.id] ?? []).length} open={openSections[`fg_${group.id}`] ?? true} onToggle={() => onToggleSection(`fg_${group.id}`)} small>
          {(friendsByGroup[group.id] ?? []).map((friend) => (
            <ContactFriendItem key={friend.id} friend={friend} userCache={userCache} onlineMap={onlineMap} onViewUser={onViewUser} onContextMenu={onFriendContext} />
          ))}
        </CollapsibleSection>
      ))}

      {(friendsByGroup[0] ?? []).length > 0 && !friendGroups.some((group) => group.id === 0) && (
        <CollapsibleSection title="默认分组" count={(friendsByGroup[0] ?? []).length} open={openSections.fg_0 ?? true} onToggle={() => onToggleSection('fg_0')} small>
          {(friendsByGroup[0] ?? []).map((friend) => (
            <ContactFriendItem key={friend.id} friend={friend} userCache={userCache} onlineMap={onlineMap} onViewUser={onViewUser} onContextMenu={onFriendContext} />
          ))}
        </CollapsibleSection>
      )}

      {friends.length === 0 && <EmptyState title="暂无好友" text="搜索账号添加好友" />}
    </>
  );
}
