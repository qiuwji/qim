import type { ConversationDTO, UserConvDTO } from '@/api/types';
import { ConversationSummaryRow } from '@/components/conversation/ConversationSummaryRow';

export function GroupListSection({
  conversations,
  details,
  mentionMap,
  onSelectChat,
}: {
  conversations: UserConvDTO[];
  details: Record<number, ConversationDTO>;
  mentionMap?: Record<number, boolean>;
  onSelectChat: (id: number) => void;
}) {
  return (
    <>
      {conversations.map((item) => (
        <ConversationSummaryRow
          key={item.conversation_id}
          conversation={item}
          detail={details[item.conversation_id]}
          subtitle={`${details[item.conversation_id]?.member_count ?? 0} 位成员`}
          mentionMap={mentionMap}
          onClick={onSelectChat}
        />
      ))}
      {conversations.length === 0 && (
        <div className="px-3 py-8 text-center text-[13px] text-[#b0b5be]">暂无群聊</div>
      )}
    </>
  );
}
