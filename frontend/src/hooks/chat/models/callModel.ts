export function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

export function callTypeLabel(type: 1 | 2): string {
  return type === 1 ? '语音通话' : '视频通话';
}

export function isCallAction(action: string): boolean {
  return ['incoming', 'calling', 'accepted', 'rejected', 'cancelled', 'ended', 'timeout', 'answered_elsewhere', 'offer', 'answer', 'ice'].includes(action);
}
