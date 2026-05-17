import React from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, UserConvDTO, UserDTO } from '@/api/types';
import { ConversationSummaryRow } from '@/components/ConversationSummaryRow';
import { ContactFriendItem } from './ContactFriendItem';

export function ContactSearchResults({
  friendsByGroup,
  friendGroups,
  groupConversations,
  details,
  userCache,
  onlineMap,
  onSelectChat,
  onViewUser,
  onFriendContext,
}: {
  friendsByGroup: Record<number, FriendDTO[]>;
  friendGroups: FriendGroupDTO[];
  groupConversations: UserConvDTO[];
  details: Record<number, ConversationDTO>;
  userCache: Record<number, UserDTO>;
  onlineMap: Record<number, boolean>;
  onSelectChat: (id: number) => void;
  onViewUser?: (uid: number) => void;
  onFriendContext: (e: React.MouseEvent, friend: FriendDTO) => void;
}) {
  const hasResults = Object.values(friendsByGroup).some((list) => list.length > 0) || groupConversations.length > 0;

  return (
    <>
      {!hasResults && <div className="px-3 py-8 text-center text-[13px] text-[#b0b5be]">未找到相关结果</div>}
      {Object.entries(friendsByGroup).map(([gid, list]) => {
        const group = friendGroups.find((item) => item.id === Number(gid));
        return (
          <div key={`sf-${gid}`}>
            <div className="px-3 py-1.5 text-xs font-semibold text-[#858c98]">{group?.name || '默认分组'}</div>
            {list.map((friend) => (
              <ContactFriendItem key={friend.id} friend={friend} userCache={userCache} onlineMap={onlineMap} onViewUser={onViewUser} onContextMenu={onFriendContext} />
            ))}
          </div>
        );
      })}
      {groupConversations.length > 0 && (
        <div>
          <div className="px-3 py-1.5 text-xs font-semibold text-[#858c98]">群聊</div>
          {groupConversations.map((item) => (
            <ConversationSummaryRow
              key={`sg-${item.conversation_id}`}
              conversation={item}
              detail={details[item.conversation_id]}
              subtitle={`${details[item.conversation_id]?.member_count ?? 0} 位成员`}
              onClick={onSelectChat}
            />
          ))}
        </div>
      )}
    </>
  );
}
