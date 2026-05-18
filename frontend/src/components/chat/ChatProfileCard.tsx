import type { ConversationDTO, UserConvDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { conversationMemberText } from '@/hooks/chat/models/conversationViewModel';
import { ConversationAvatar } from '@/components/conversation/ConversationAvatar';
import { SwitchRow } from '@/components/ui';

export function ChatProfileCard({ chat, detail, members, isAdmin, onTogglePin, onToggleMute, onHide, onUploadGroupAvatar }: {
  chat: UserConvDTO; detail?: ConversationDTO; members: { uid: number }[]; isAdmin: boolean;
  onTogglePin: (convID: number) => void; onToggleMute: (convID: number) => void; onHide: (convID: number) => void; onUploadGroupAvatar: (file: File) => void;
}) {
  return (
    <div className="detail-card chat-profile-card">
      <ConversationAvatar conversation={chat} detail={detail} large />
      <strong>{chatTitle(chat, detail)}</strong>
      <span>{conversationMemberText(detail, members.length)}</span>
      {isAdmin && (
        <label className="upload-btn small">
          上传群头像
          <input type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onUploadGroupAvatar(e.target.files[0])} />
        </label>
      )}
      <div className="switch-list">
        <SwitchRow label="置顶聊天" on={chat.is_pinned} onClick={() => onTogglePin(chat.conversation_id)} />
        <SwitchRow label="消息免打扰" on={chat.is_muted} onClick={() => onToggleMute(chat.conversation_id)} />
      </div>
      <button className="wide-btn danger-soft" onClick={() => onHide(chat.conversation_id)}>从聊天列表移除</button>
    </div>
  );
}
