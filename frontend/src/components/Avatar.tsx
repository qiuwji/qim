import type { UserDTO } from '../api/types';
import { avatarURL, shortName } from '../utils';

export function Avatar({ user, large = false, online, badge, className, onClick }: { user: UserDTO; large?: boolean; online?: boolean; badge?: number; className?: string; onClick?: () => void }) {
  const url = avatarURL(user.avatar);
  const cls = `avatar ${large ? 'large' : ''} ${className ?? ''}`.trim();
  const inner = url
    ? <img className={cls} src={url} alt={user.nickname} onClick={onClick} />
    : <div className={`${cls} fallback`} onClick={onClick}>{shortName(user.nickname || user.username)}</div>;

  if (online === undefined && !badge) return inner;

  return (
    <div className="avatar-wrap" onClick={onClick}>
      {inner}
      {online !== undefined && <span className={`status-dot ${online ? 'online' : 'offline'}`} />}
      {badge !== undefined && badge > 0 && <b className="avatar-badge">{badge > 99 ? '99+' : badge}</b>}
    </div>
  );
}
