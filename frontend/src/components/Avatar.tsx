import type { UserDTO } from '../api/types';
import { avatarURL, shortName } from '../utils';

export function Avatar({ user, large = false, small = false, online, badge, className, onClick, onMouseEnter, onMouseLeave }: { user: UserDTO; large?: boolean; small?: boolean; online?: boolean; badge?: number; className?: string; onClick?: (e: React.MouseEvent) => void; onMouseEnter?: (e: React.MouseEvent) => void; onMouseLeave?: (e: React.MouseEvent) => void }) {
  const url = avatarURL(user.avatar);
  const cls = `avatar ${large ? 'large' : ''} ${small ? 'small' : ''} ${className ?? ''}`.trim();
  const inner = url
    ? <img className={cls} src={url} alt={user.nickname} onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave} />
    : <div className={`${cls} fallback`} onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>{shortName(user.nickname || user.username)}</div>;

  if (online === undefined && !badge) return inner;

  return (
    <div className="avatar-wrap" onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
      {inner}
      {online !== undefined && <span className={`status-dot ${online ? 'online' : 'offline'}`} />}
      {badge !== undefined && badge > 0 && <b className="avatar-badge">{badge > 99 ? '99+' : badge}</b>}
    </div>
  );
}
