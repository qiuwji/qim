import { useCallback, useLayoutEffect, useRef } from 'react';

const NEAR_BOTTOM_OFFSET = 120;
const LOAD_MORE_OFFSET = 80;

export function useChatScroll({
  conversationID,
  messagesLength,
  latestMessageKey,
  hasMore,
  onLoadMore,
}: {
  conversationID?: number;
  messagesLength: number;
  latestMessageKey: string;
  hasMore: boolean;
  onLoadMore: () => void | Promise<void>;
}) {
  const areaRef = useRef<HTMLDivElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const prevLenRef = useRef(0);
  const prevLatestMessageKeyRef = useRef('');
  const loadingMoreRef = useRef(false);
  const pendingInitialScrollRef = useRef(false);
  const stickToBottomRef = useRef(true);

  const isNearBottom = useCallback(() => {
    const el = areaRef.current;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < NEAR_BOTTOM_OFFSET;
  }, []);

  const scrollToBottom = useCallback((smooth: boolean) => {
    const el = areaRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
    stickToBottomRef.current = true;
    if (smooth) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, []);

  const captureStickToBottom = useCallback(() => {
    stickToBottomRef.current = isNearBottom();
  }, [isNearBottom]);

  const triggerLoadMore = useCallback(() => {
    const el = areaRef.current;
    if (!el || !hasMore || loadingMoreRef.current) return;
    const prevHeight = el.scrollHeight;
    const prevTop = el.scrollTop;
    loadingMoreRef.current = true;
    Promise.resolve(onLoadMore()).finally(() => {
      requestAnimationFrame(() => {
        const latest = areaRef.current;
        if (latest) {
          latest.scrollTop = latest.scrollHeight - prevHeight + prevTop;
        }
        loadingMoreRef.current = false;
      });
    });
  }, [hasMore, onLoadMore]);

  const handleAreaScroll = useCallback(() => {
    const el = areaRef.current;
    captureStickToBottom();
    if (el && el.scrollTop <= LOAD_MORE_OFFSET) triggerLoadMore();
  }, [captureStickToBottom, triggerLoadMore]);

  useLayoutEffect(() => {
    pendingInitialScrollRef.current = true;
    loadingMoreRef.current = false;
    scrollToBottom(false);
  }, [conversationID, scrollToBottom]);

  useLayoutEffect(() => {
    if (pendingInitialScrollRef.current) {
      scrollToBottom(false);
      requestAnimationFrame(() => scrollToBottom(false));
      if (messagesLength > 0 || !conversationID) {
        pendingInitialScrollRef.current = false;
      }
      prevLenRef.current = messagesLength;
      prevLatestMessageKeyRef.current = latestMessageKey;
      return;
    }
    if (loadingMoreRef.current) {
      loadingMoreRef.current = false;
      prevLenRef.current = messagesLength;
      prevLatestMessageKeyRef.current = latestMessageKey;
      return;
    }
    const hasNewTailMessage = latestMessageKey !== prevLatestMessageKeyRef.current;
    if ((messagesLength > prevLenRef.current || hasNewTailMessage) && stickToBottomRef.current) {
      scrollToBottom(true);
      requestAnimationFrame(() => scrollToBottom(false));
    }
    prevLenRef.current = messagesLength;
    prevLatestMessageKeyRef.current = latestMessageKey;
  }, [conversationID, latestMessageKey, messagesLength, scrollToBottom]);

  useLayoutEffect(() => {
    const content = contentRef.current;
    if (!content || typeof ResizeObserver === 'undefined') return undefined;
    const observer = new ResizeObserver(() => {
      if (stickToBottomRef.current && !loadingMoreRef.current) {
        scrollToBottom(false);
      }
    });
    observer.observe(content);
    return () => observer.disconnect();
  }, [scrollToBottom]);

  return {
    areaRef,
    contentRef,
    bottomRef,
    captureStickToBottom,
    handleAreaScroll,
  };
}
