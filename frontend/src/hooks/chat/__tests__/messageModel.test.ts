import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { MessageDTO, UserConvDTO } from '@/api/types';
import {
  applyMessagePreview,
  applyPreviewTexts,
  callRecordSummary,
  deletedMessageStorageKey,
  endReasonText,
  isCallRecord,
  isSystemMessage,
  latestVisibleMessage,
  mergeLoadedMessages,
  messageDisplayText,
  messagePreviewText,
  MSG_TYPE_CALL_RECORD,
  MSG_TYPE_IMAGE,
  MSG_TYPE_LEGACY_SYSTEM,
  MSG_TYPE_SYSTEM,
  parseCallRecordContent,
  persistDeletedMessageID,
  readDeletedMessageIDs,
  removeMessageByID,
  revokedPreviewUpdate,
  visibleMessages,
} from '../models/messageModel';

function createStorage() {
  const store = new Map<string, string>();
  return {
    getItem: vi.fn((key: string) => store.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => { store.set(key, value); }),
    removeItem: vi.fn((key: string) => { store.delete(key); }),
    clear: vi.fn(() => { store.clear(); }),
  };
}

function message(id: number, overrides: Partial<MessageDTO> = {}): MessageDTO {
  return {
    id,
    conversation_id: 1,
    seq: id,
    sender_id: 2,
    msg_type: 1,
    content: `message-${id}`,
    reply_to: 0,
    client_id: '',
    created_at: 100 + id,
    ...overrides,
  };
}

function conversation(id: number): UserConvDTO {
  return {
    conversation_id: id,
    is_pinned: false,
    is_muted: false,
    unread_count: 0,
    last_msg_at: 1,
  };
}

describe('messageModel', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', createStorage());
  });

  it('读写本地删除消息 ID，兼容旧的 conversation:message 格式', () => {
    const key = deletedMessageStorageKey(123);
    localStorage.setItem(key, JSON.stringify([1, '2:20', 'bad', 0]));

    expect(readDeletedMessageIDs(key)).toEqual([1, 20]);

    const deleted = new Set<number>([1]);
    persistDeletedMessageID(key, deleted, 3);
    expect([...deleted]).toEqual([1, 3]);
    expect(JSON.parse(localStorage.getItem(key) ?? '[]')).toEqual([1, 3]);
  });

  it('本地删除消息配置损坏时返回空数组', () => {
    const key = deletedMessageStorageKey(123);
    localStorage.setItem(key, '{bad json');

    expect(readDeletedMessageIDs(key)).toEqual([]);
  });

  it('生成消息预览并识别撤回消息', () => {
    expect(messagePreviewText(undefined)).toBeUndefined();
    expect(messagePreviewText(message(1, { content: '' }))).toBeUndefined();
    expect(messagePreviewText(message(2, { content: 'hello' }))).toBe('hello');
    expect(messagePreviewText(message(3, { revoked: true }))).toBe('消息已撤回');
    expect(messagePreviewText(message(4, { msg_type: MSG_TYPE_IMAGE, content: '/uploads/img.png' }))).toBe('[图片]');
  });

  it('messageDisplayText 对图片消息返回 [图片]', () => {
    expect(messageDisplayText(message(1, { msg_type: MSG_TYPE_IMAGE, content: '/uploads/abc.png' }))).toBe('[图片]');
    expect(messageDisplayText(message(2, { content: 'hello' }))).toBe('hello');
    expect(messageDisplayText(message(3, { msg_type: MSG_TYPE_IMAGE, content: '' }))).toBe('[图片]');
  });

  it('同时识别当前和历史系统消息类型', () => {
    expect(isSystemMessage(message(1, { msg_type: MSG_TYPE_SYSTEM }))).toBe(true);
    expect(isSystemMessage(message(2, { msg_type: MSG_TYPE_LEGACY_SYSTEM }))).toBe(true);
    expect(isSystemMessage(message(3, { msg_type: 1 }))).toBe(false);
  });

  it('过滤本地删除消息并返回最新可见消息', () => {
    const list = [message(1), message(2), message(3)];
    const deleted = new Set([2]);

    expect(visibleMessages(list, deleted).map((item) => item.id)).toEqual([1, 3]);
    expect(latestVisibleMessage(list, deleted)?.id).toBe(1);
    expect(removeMessageByID(list, 2).map((item) => item.id)).toEqual([1, 3]);
  });

  it('更新单个会话预览，空消息会删除旧预览', () => {
    expect(applyMessagePreview({ 1: 'old' }, 1, message(1, { content: 'new' }))).toEqual({ 1: 'new' });
    expect(applyMessagePreview({ 1: 'old' }, 1, undefined)).toEqual({});
    expect(applyMessagePreview({ 1: 'old' }, 1, message(2, { content: '' }))).toEqual({});
  });

  it('批量更新会话预览，未提供预览时清理旧值', () => {
    const prev = { 1: 'old-1', 2: 'old-2', 3: 'keep' };
    const next = applyPreviewTexts(prev, [conversation(1), conversation(2)], { 1: 'new-1', 2: undefined });

    expect(next).toEqual({ 1: 'new-1', 3: 'keep' });
  });

  it('加载消息按 seq 排序，翻页时拼接到已有消息前方', () => {
    const loaded = [message(3), message(1), message(2)];
    const existing = [message(4)];

    expect(mergeLoadedMessages(loaded, existing, 0).map((item) => item.id)).toEqual([1, 2, 3]);
    expect(mergeLoadedMessages(loaded, existing, 4).map((item) => item.id)).toEqual([1, 2, 3, 4]);
  });

  it('撤回非最新消息时不更新预览', () => {
    const list = [message(1), message(2)];

    expect(revokedPreviewUpdate(list, 1, false, new Set())).toEqual({ kind: 'none' });
  });

  it('撤回本地已删除消息时回退到上一条可见消息', () => {
    const list = [message(1), message(2)];

    expect(revokedPreviewUpdate(list, 2, true, new Set([2]))).toEqual({
      kind: 'message',
      message: message(1),
    });
  });

  it('撤回最新未删除消息时显示撤回预览文案', () => {
    const list = [message(1), message(2)];

    expect(revokedPreviewUpdate(list, 2, true, new Set())).toEqual({
      kind: 'text',
      text: '消息已撤回',
    });
  });

  describe('通话记录消息', () => {
    it('isCallRecord 识别 type 6', () => {
      expect(isCallRecord(message(1, { msg_type: MSG_TYPE_CALL_RECORD }))).toBe(true);
      expect(isCallRecord(message(2, { msg_type: 1 }))).toBe(false);
      expect(isCallRecord(message(3, { msg_type: MSG_TYPE_SYSTEM }))).toBe(false);
    });

    it('parseCallRecordContent 解析合法 JSON', () => {
      const json = JSON.stringify({ call_id: 'c1', call_type: 1, duration: 120, end_reason: 'hangup', status: 2 });
      const result = parseCallRecordContent(json);
      expect(result).not.toBeNull();
      expect(result!.call_id).toBe('c1');
      expect(result!.call_type).toBe(1);
      expect(result!.duration).toBe(120);
      expect(result!.end_reason).toBe('hangup');
      expect(result!.status).toBe(2);
    });

    it('parseCallRecordContent 缺失字段使用默认值', () => {
      const result = parseCallRecordContent(JSON.stringify({ call_type: 2, status: 1 }));
      expect(result).not.toBeNull();
      expect(result!.call_id).toBe('');
      expect(result!.duration).toBe(0);
      expect(result!.end_reason).toBe('');
      expect(result!.status).toBe(1);
    });

    it('parseCallRecordContent 无效 JSON 返回 null', () => {
      expect(parseCallRecordContent('invalid')).toBeNull();
      expect(parseCallRecordContent('')).toBeNull();
    });

    it('parseCallRecordContent 非法 call_type 返回 null', () => {
      expect(parseCallRecordContent(JSON.stringify({ call_type: 0 }))).toBeNull();
      expect(parseCallRecordContent(JSON.stringify({ call_type: 3 }))).toBeNull();
    });

    it('endReasonText 映射结束原因', () => {
      expect(endReasonText('rejected')).toBe('已拒绝');
      expect(endReasonText('timeout')).toBe('无人接听');
      expect(endReasonText('cancelled')).toBe('已取消');
      expect(endReasonText('disconnect')).toBe('连接断开');
      expect(endReasonText('hangup')).toBe('');
      expect(endReasonText('')).toBe('');
    });

    it('callRecordSummary 已接通显示时长', () => {
      const json = JSON.stringify({ call_id: 'c1', call_type: 1, duration: 125, end_reason: 'hangup', status: 2 });
      expect(callRecordSummary(json)).toBe('[语音通话] 02:05');
    });

    it('callRecordSummary 视频已接通', () => {
      const json = JSON.stringify({ call_id: 'c2', call_type: 2, duration: 60, end_reason: 'hangup', status: 2 });
      expect(callRecordSummary(json)).toBe('[视频通话] 01:00');
    });

    it('callRecordSummary 未接通显示原因', () => {
      const json = JSON.stringify({ call_id: 'c3', call_type: 1, duration: 0, end_reason: 'rejected', status: 1 });
      expect(callRecordSummary(json)).toBe('[语音通话] 已拒绝');
    });

    it('callRecordSummary 超时未接通', () => {
      const json = JSON.stringify({ call_id: 'c4', call_type: 2, duration: 0, end_reason: 'timeout', status: 1 });
      expect(callRecordSummary(json)).toBe('[视频通话] 无人接听');
    });

    it('callRecordSummary 无效内容返回默认', () => {
      expect(callRecordSummary('invalid')).toBe('[通话记录]');
      expect(callRecordSummary('')).toBe('[通话记录]');
    });

    it('messageDisplayText type 6 返回摘要文案', () => {
      const completed = message(1, {
        msg_type: MSG_TYPE_CALL_RECORD,
        content: JSON.stringify({ call_id: 'c1', call_type: 1, duration: 120, end_reason: 'hangup', status: 2 }),
      });
      expect(messageDisplayText(completed)).toBe('[语音通话] 02:00');

      const missed = message(2, {
        msg_type: MSG_TYPE_CALL_RECORD,
        content: JSON.stringify({ call_id: 'c2', call_type: 2, duration: 0, end_reason: 'cancelled', status: 1 }),
      });
      expect(messageDisplayText(missed)).toBe('[视频通话] 已取消');
    });
  });
});
