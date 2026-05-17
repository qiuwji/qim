import type { ConversationDTO, UserConvDTO } from '@/api/types';
import { chatTitle, shortName } from '@/utils';
import { SwitchRow } from '@/components/ui';

export function ChatProfileCard({ chat, detail, members, isAdmin, onTogglePin, onToggleMute, onUploadGroupAvatar }: {
  chat: UserConvDTO; detail?: ConversationDTO; members: { uid: number }[]; isAdmin: boolean;
  onTogglePin: (convID: number) => void; onToggleMute: (convID: number) => void; onUploadGroupAvatar: (file: File) => void;
}) {
  const isGroup = detail?.type === 2;
  return (
    <div className="detail-card chat-profile-card">
      <div className="detail-avatar">{shortName(chatTitle(chat, detail))}</div>
      <strong>{chatTitle(chat, detail)}</strong>
      <span>{isGroup ? `${members.length} 位成员` : '私聊'}</span>
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
    </div>
  );
}