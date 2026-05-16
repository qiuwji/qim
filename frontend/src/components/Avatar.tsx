import type { UserDTO } from '../api/types';
import { avatarURL, shortName } from '../utils';

export function Avatar({ user, large = false, small = false, online, badge, className, onClick, onMouseEnter, onMouseLeave }: { user: UserDTO; large?: boolean; small?: boolean; online?: boolean; badge?: number; className?: string; onClick?: (e: React.MouseEvent) => void; onMouseEnter?: (e: React.MouseEvent) => void; onMouseLeave?: (e: React.MouseEvent) => void }) {
  const url = avatarURL(user.avatar);
  const sizeClass = large ? 'h-[72px] w-[72px] rounded-2xl text-2xl' : small ? 'h-9 w-9 rounded-lg text-xs' : 'h-[42px] w-[42px] rounded-[10px] text-sm';
  const cls = `${sizeClass} shrink-0 object-cover ${className ?? ''}`.trim();
  const inner = url
    ? <img className={cls} src={url} alt={user.nickname} onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave} />
    : <div className={`${cls} grid place-items-center bg-gradient-to-br from-[#43a047] to-[#07c160] font-bold text-white`} onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>{shortName(user.nickname || user.username)}</div>;

  if (online === undefined && !badge) return inner;

  return (
    <div className="relative inline-flex shrink-0" onClick={onClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
      {inner}
      {online !== undefined && <span className={`absolute right-0 bottom-0 h-2.5 w-2.5 rounded-full border-2 border-white ${online ? 'bg-[#07c160]' : 'bg-[#b8c0cc]'}`} />}
      {badge !== undefined && badge > 0 && <b className="absolute -top-1 -right-1 grid h-[18px] min-w-[18px] place-items-center rounded-full bg-[#f04b45] px-1 text-[10px] font-semibold leading-none text-white">{badge > 99 ? '99+' : badge}</b>}
    </div>
  );
}
