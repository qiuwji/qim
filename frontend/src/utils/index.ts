export function currentSecond() {
  return Math.floor(Date.now() / 1000);
}

export function validatePassword(pwd: string): string | null {
  if (pwd.length < 8 || pwd.length > 20) return '密码长度需为 8-20 位';
  if (!/[A-Z]/.test(pwd)) return '密码需包含大写字母';
  if (!/[a-z]/.test(pwd)) return '密码需包含小写字母';
  if (!/[^A-Za-z0-9]/.test(pwd)) return '密码需包含特殊字符';
  return null;
}

export function timeText(value?: number) {
  if (!value) return '';
  const target = new Date(value * 1000);
  const now = new Date();
  const diffDays = calendarDayDiff(now, target);
  const hm = hourMinuteText(target);
  if (diffDays === 0) return hm;
  if (diffDays === 1) return `昨天 ${hm}`;
  if (diffDays === 2) return `前天 ${hm}`;
  if (diffDays > 2 && diffDays < 7) return `${weekdayText(target)} ${hm}`;
  return `${dateText(target)} ${hm}`;
}

function calendarDayDiff(now: Date, target: Date) {
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const targetDay = new Date(target.getFullYear(), target.getMonth(), target.getDate()).getTime();
  return Math.floor((today - targetDay) / 86400000);
}

function hourMinuteText(date: Date) {
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false });
}

function weekdayText(date: Date) {
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()];
}

function dateText(date: Date) {
  return [
    date.getFullYear(),
    String(date.getMonth() + 1).padStart(2, '0'),
    String(date.getDate()).padStart(2, '0'),
  ].join('/');
}

export function shortName(name?: string) {
  return (name || '?').trim().slice(0, 1).toUpperCase();
}

export function avatarURL(url?: string) {
  if (!url) return '';
  if (/^https?:\/\//.test(url)) return url;
  return url;
}

export function upsertMessage(list: import('../api/types').MessageDTO[], next: import('../api/types').MessageDTO) {
  const i = list.findIndex((item) => item.id === next.id || (!!next.client_id && item.client_id === next.client_id));
  if (i >= 0) {
    const c = [...list];
    c[i] = { ...c[i], ...next };
    return c.sort((a, b) => a.seq - b.seq || a.created_at - b.created_at);
  }
  return [...list, next].sort((a, b) => a.seq - b.seq || a.created_at - b.created_at);
}

export function chatTitle(conv: import('../api/types').UserConvDTO, detail?: import('../api/types').ConversationDTO) {
  if (detail?.name) return detail.name;
  return `聊天 ${conv.conversation_id}`;
}

export function displayName(uid: number, userCache: Record<number, import('../api/types').UserDTO>, friendMap?: Record<number, import('../api/types').FriendDTO>) {
  const friend = friendMap?.[uid];
  if (friend?.remark) return friend.remark;
  const user = userCache[uid];
  return user?.nickname || user?.username || `用户 ${uid}`;
}

export function pushText(msg: import('../api/types').WsResponse) {
  const map: Record<string, string> = {
    'friend/request': '收到新的好友申请',
    'friend/accepted': '好友申请已通过',
    'friend/rejected': '好友申请被拒绝',
    'member/joined': '群成员已加入',
    'member/left': '群成员已退出',
    'member/kicked': '群成员已移除',
    'member/owner_transferred': '群主已转让',
    'conversation/updated': '聊天信息已更新',
    'conversation/group_dissolved': '群聊已解散',
  };
  return map[`${msg.type}/${msg.action}`] ?? '收到新的实时事件';
}

export function scrollToMessage(messageID: number, highlightClass = 'highlight-msg', duration = 2000) {
  const el = document.getElementById(`msg-${messageID}`);
  if (!el) return;
  el.scrollIntoView({ behavior: 'smooth', block: 'center' });
  el.classList.add(highlightClass);
  setTimeout(() => el.classList.remove(highlightClass), duration);
}
