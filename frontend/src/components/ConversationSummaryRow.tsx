import type { ConversationDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { ConversationAvatar } from '@/components/ConversationAvatar';

export function ConversationSummaryRow({ conversation, detail, subtitle, members, currentUID, userCache, onlineMap, selected = false, onClick }: {
  conversation: UserConvDTO;
  detail?: ConversationDTO;
  subtitle?: string;
  members?: MemberDTO[];
  currentUID?: number;
  userCache?: Record<number, UserDTO>;
  onlineMap?: Record<number, boolean>;
  selected?: boolean;
  onClick: (id: number) => void;
}) {
  return (
    <button className={`flex w-full items-center gap-3 border-b border-[#f0f1f3] px-3 py-2.5 text-left transition hover:bg-[#e8e8e8] ${selected ? 'bg-[#d4edda]' : ''}`} onClick={() => onClick(conversation.conversation_id)}>
      <ConversationAvatar conversation={conversation} detail={detail} members={members} currentUID={currentUID} userCache={userCache} onlineMap={onlineMap} small />
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-[#1a1a1a]">{chatTitle(conversation, detail)}</div>
        {subtitle && <div className="text-xs text-[#999]">{subtitle}</div>}
      </div>
      {selected && <span className="text-sm font-semibold text-[#1aad19]">✓</span>}
    </button>
  );
}
