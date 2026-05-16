import type { MessageDTO, UserDTO } from '../api/types';
import { Avatar } from './Avatar';
import { timeText } from '../utils';

export function MessageBubble({ message, mine, senderName, senderUser, replySource, onContextMenu, onAvatarEnter, onAvatarLeave, onAvatarClick }: {
  message: MessageDTO; mine: boolean; senderName: string; senderUser?: UserDTO; replySource: MessageDTO | undefined;
  onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void;
  onAvatarEnter?: (e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (e: React.MouseEvent) => void;
}) {
  const isImage = message.msg_type === 2 && !message.revoked;

  if (message.revoked) {
    return (
      <div className="revoke-notice">
        {mine ? '你' : senderName}撤回了一条消息
      </div>
    );
  }

  return (
    <div className={`message-row ${mine ? 'mine' : ''}`} onContextMenu={(e) => onContextMenu(e, message)}>
      {senderUser
        ? <Avatar user={senderUser} className="mini-avatar avatar-interactive" onClick={onAvatarClick} onMouseEnter={onAvatarEnter} onMouseLeave={onAvatarLeave} />
        : <div className="mini-avatar avatar-interactive" onMouseEnter={onAvatarEnter} onMouseLeave={onAvatarLeave} onClick={onAvatarClick}>?</div>
      }
      <div className="bubble">
        <span className="bubble-meta">{senderName} · {timeText(message.created_at)}</span>
        {replySource && <div className="reply-quote">↩ {replySource.content.slice(0, 50)}{replySource.content.length > 50 ? '...' : ''}</div>}
        <div className="bubble-line">
          {isImage
            ? <p><img className="chat-image" src={message.content} alt="图片" onClick={() => window.open(message.content, '_blank')} /></p>
            : <p>{message.content}</p>}
        </div>
      </div>
    </div>
  );
}
