import { useCallback, useLayoutEffect, useRef } from 'react';

const NEAR_BOTTOM_OFFSET = 120;
const LOAD_MORE_OFFSET = 80;

export function useChatScroll({
  conversationID,
  messagesLength,
  hasMore,
  onLoadMore,
}: {
  conversationID?: number;
  messagesLength: number;
  hasMore: boolean;
  onLoadMore: () => void | Promise<void>;
}) {
  const areaRef = useRef<HTMLDivElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const prevLenRef = useRef(0);
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
      return;
    }
    if (loadingMoreRef.current) {
      loadingMoreRef.current = false;
      prevLenRef.current = messagesLength;
      return;
    }
    if (messagesLength > prevLenRef.current && stickToBottomRef.current) {
      scrollToBottom(true);
    }
    prevLenRef.current = messagesLength;
  }, [conversationID, messagesLength, scrollToBottom]);

  return {
    areaRef,
    bottomRef,
    captureStickToBottom,
    handleAreaScroll,
  };
}
