import { afterEach, describe, expect, it, vi } from 'vitest';
import { timeText } from './index';

function seconds(date: string) {
  return Math.floor(new Date(date).getTime() / 1000);
}

describe('timeText', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('按聊天时间规则展示 24 小时时间', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-17T13:05:00+08:00'));

    expect(timeText(seconds('2026-05-17T09:08:00+08:00'))).toBe('09:08');
    expect(timeText(seconds('2026-05-16T23:59:00+08:00'))).toBe('昨天 23:59');
    expect(timeText(seconds('2026-05-15T00:01:00+08:00'))).toBe('前天 00:01');
    expect(timeText(seconds('2026-05-14T18:30:00+08:00'))).toBe('周四 18:30');
    expect(timeText(seconds('2026-05-10T12:00:00+08:00'))).toBe('2026/05/10 12:00');
  });

  it('空时间返回空字符串', () => {
    expect(timeText()).toBe('');
    expect(timeText(0)).toBe('');
  });
});
