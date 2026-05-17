import { afterEach, describe, expect, it, vi } from 'vitest';
import { timeText } from './index';

function seconds(year: number, month: number, day: number, hour: number, minute: number) {
  return Math.floor(new Date(year, month - 1, day, hour, minute).getTime() / 1000);
}

describe('timeText', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('按聊天时间规则展示 24 小时时间', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 4, 17, 13, 5));

    expect(timeText(seconds(2026, 5, 17, 9, 8))).toBe('09:08');
    expect(timeText(seconds(2026, 5, 16, 23, 59))).toBe('昨天 23:59');
    expect(timeText(seconds(2026, 5, 15, 0, 1))).toBe('前天 00:01');
    expect(timeText(seconds(2026, 5, 14, 18, 30))).toBe('周四 18:30');
    expect(timeText(seconds(2026, 5, 10, 12, 0))).toBe('2026/05/10 12:00');
  });

  it('空时间返回空字符串', () => {
    expect(timeText()).toBe('');
    expect(timeText(0)).toBe('');
  });
});
