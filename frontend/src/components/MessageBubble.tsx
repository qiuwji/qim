import type { MessageDTO, UserDTO } from '@/api/types';
import { isSystemMessage } from '@/hooks/chat/models/messageModel';
import { Avatar } from '@/components/ui';
import { timeText } from '@/utils';

export function MessageBubble({ message, mine, senderName, senderUser, replySource, onContextMenu, onAvatarEnter, onAvatarLeave, onAvatarClick }: {
  message: MessageDTO; mine: boolean; senderName: string; senderUser?: UserDTO; replySource: MessageDTO | undefined;
  onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void;
  onAvatarEnter?: (e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (e: React.MouseEvent) => void;
}) {
  const isImage = message.msg_type === 2 && !message.revoked;

  if (isSystemMessage(message) && !message.revoked) {
    return <div className="mx-auto my-3 w-fit max-w-[min(520px,78%)] rounded-full bg-[#eef0f3] px-2.5 py-1.5 text-center text-xs leading-normal text-[#7b8491]">{message.content}</div>;
  }

  if (message.revoked) {
    return (
      <div className="my-1 py-1.5 text-center text-xs leading-normal text-[#9aa1ad]">
        {mine ? '你' : senderName}撤回了一条消息
      </div>
    );
  }

  return (
    <div className={`my-3.5 flex items-start gap-2.5 ${mine ? 'flex-row-reverse' : ''}`} onContextMenu={(e) => onContextMenu(e, message)}>
      {senderUser
        ? <Avatar user={senderUser} small className="avatar-interactive cursor-pointer" onClick={onAvatarClick} onMouseEnter={onAvatarEnter} onMouseLeave={onAvatarLeave} />
        : <div className="avatar-interactive grid h-[34px] w-[34px] cursor-pointer place-items-center rounded-lg bg-gradient-to-br from-[#5b8def] to-[#4080e0] text-xs font-bold text-white" onMouseEnter={onAvatarEnter} onMouseLeave={onAvatarLeave} onClick={onAvatarClick}>?</div>
      }
      <div className={`grid max-w-[min(560px,72%)] gap-1.5 ${mine ? 'justify-items-end' : ''}`}>
        <span className="text-xs text-[#9aa1ad]">{senderName} · {timeText(message.created_at)}</span>
        {replySource && <div className="mb-0.5 rounded-r-md border-l-[3px] border-[#12b35f] bg-[#f0faf4] px-2.5 py-1.5 text-[13px] text-[#555f6d]">↩ {replySource.content.slice(0, 50)}{replySource.content.length > 50 ? '...' : ''}</div>}
        <div className={`flex items-end gap-2 ${mine ? 'flex-row-reverse' : ''}`}>
          {isImage
            ? <p className="m-0 rounded-[10px] border border-[#dfe3e8] bg-white px-3.5 py-2.5 leading-[1.55] shadow-[0_1px_2px_rgba(31,35,41,0.04)]"><img className="max-h-[200px] max-w-[240px] cursor-pointer rounded-md hover:opacity-90" src={message.content} alt="图片" onClick={() => window.open(message.content, '_blank')} /></p>
            : <p className={`m-0 whitespace-pre-wrap break-words rounded-[10px] border px-3.5 py-2.5 leading-[1.55] shadow-[0_1px_2px_rgba(31,35,41,0.04)] ${mine ? 'border-[#b8e7a9] bg-[#95ec69] text-[#152b12]' : 'border-[#dfe3e8] bg-white text-[#1f2329]'}`}>{message.content}</p>}
        </div>
      </div>
    </div>
  );
}
