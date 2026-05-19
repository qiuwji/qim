import { describe, it, expect } from 'vitest';
import { formatDuration, callTypeLabel, isCallAction } from '../models/callModel';

describe('formatDuration', () => {
  it('formats 0 seconds', () => {
    expect(formatDuration(0)).toBe('00:00');
  });

  it('formats 59 seconds', () => {
    expect(formatDuration(59)).toBe('00:59');
  });

  it('formats 60 seconds as 01:00', () => {
    expect(formatDuration(60)).toBe('01:00');
  });

  it('formats 3600 seconds as 60:00', () => {
    expect(formatDuration(3600)).toBe('60:00');
  });

  it('formats large duration', () => {
    expect(formatDuration(3661)).toBe('61:01');
  });
});

describe('callTypeLabel', () => {
  it('returns voice for type 1', () => {
    expect(callTypeLabel(1)).toBe('语音通话');
  });

  it('returns video for type 2', () => {
    expect(callTypeLabel(2)).toBe('视频通话');
  });
});

describe('isCallAction', () => {
  it('recognizes incoming as call action', () => {
    expect(isCallAction('incoming')).toBe(true);
  });

  it('recognizes accepted as call action', () => {
    expect(isCallAction('accepted')).toBe(true);
  });

  it('recognizes ended as call action', () => {
    expect(isCallAction('ended')).toBe(true);
  });

  it('recognizes offer as call action', () => {
    expect(isCallAction('offer')).toBe(true);
  });

  it('recognizes answer as call action', () => {
    expect(isCallAction('answer')).toBe(true);
  });

  it('recognizes ice as call action', () => {
    expect(isCallAction('ice')).toBe(true);
  });

  it('returns false for non-call actions', () => {
    expect(isCallAction('send')).toBe(false);
    expect(isCallAction('new')).toBe(false);
    expect(isCallAction('list')).toBe(false);
  });
});
