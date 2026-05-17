import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useChatScroll } from '../useChatScroll';

type Ref<T> = { current: T };
type Effect = () => void | (() => void);

const reactHooks = vi.hoisted(() => {
  const state = {
    refCursor: 0,
    callbackCursor: 0,
    effectCursor: 0,
    refs: [] as Ref<unknown>[],
    callbacks: [] as unknown[],
    callbackDeps: [] as unknown[][],
    effectDeps: [] as unknown[][],
    effects: [] as Effect[],
  };
  function depsChanged(prev: unknown[] | undefined, next: unknown[] | undefined) {
    if (!prev || !next) return true;
    return prev.length !== next.length || prev.some((item, index) => item !== next[index]);
  }
  return {
    state,
    resetRender() {
      state.refCursor = 0;
      state.callbackCursor = 0;
      state.effectCursor = 0;
      state.effects = [];
    },
    resetAll() {
      state.refCursor = 0;
      state.callbackCursor = 0;
      state.effectCursor = 0;
      state.refs = [];
      state.callbacks = [];
      state.callbackDeps = [];
      state.effectDeps = [];
      state.effects = [];
    },
    useRef: vi.fn((initial: unknown) => {
      const index = state.refCursor;
      state.refCursor += 1;
      if (!state.refs[index]) state.refs[index] = { current: initial };
      return state.refs[index];
    }),
    useCallback: vi.fn((fn: unknown, deps?: unknown[]) => {
      const index = state.callbackCursor;
      state.callbackCursor += 1;
      if (depsChanged(state.callbackDeps[index], deps)) {
        state.callbacks[index] = fn;
        state.callbackDeps[index] = deps ?? [];
      }
      return state.callbacks[index];
    }),
    useLayoutEffect: vi.fn((effect: Effect, deps?: unknown[]) => {
      const index = state.effectCursor;
      state.effectCursor += 1;
      if (depsChanged(state.effectDeps[index], deps)) {
        state.effects.push(effect);
        state.effectDeps[index] = deps ?? [];
      }
    }),
  };
});

vi.mock('react', () => ({
  useCallback: reactHooks.useCallback,
  useLayoutEffect: reactHooks.useLayoutEffect,
  useRef: reactHooks.useRef,
}));

class ResizeObserverMock {
  static callbacks: ResizeObserverCallback[] = [];

  constructor(callback: ResizeObserverCallback) {
    ResizeObserverMock.callbacks.push(callback);
  }

  observe = vi.fn();
  disconnect = vi.fn();
}

function renderUseChatScroll(props: {
  conversationID?: number;
  messagesLength: number;
  latestMessageKey: string;
  hasMore?: boolean;
  onLoadMore?: () => void | Promise<void>;
}) {
  reactHooks.resetRender();
  // Test harness mocks React hooks and invokes the target hook directly.
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const result = useChatScroll({
    hasMore: false,
    onLoadMore: vi.fn(),
    ...props,
  });
  const [areaRef, contentRef, bottomRef] = reactHooks.state.refs as [
    Ref<FakeScrollArea | null>,
    Ref<object | null>,
    Ref<{ scrollIntoView: ReturnType<typeof vi.fn> } | null>,
  ];
  return { result, areaRef, contentRef, bottomRef };
}

function flushLayoutEffects() {
  const effects = [...reactHooks.state.effects];
  reactHooks.state.effects = [];
  for (const effect of effects) effect();
}

function createScrollArea(overrides: Partial<FakeScrollArea> = {}): FakeScrollArea {
  return {
    scrollHeight: 1000,
    scrollTop: 700,
    clientHeight: 300,
    ...overrides,
  };
}

type FakeScrollArea = {
  scrollHeight: number;
  scrollTop: number;
  clientHeight: number;
};

describe('useChatScroll', () => {
  beforeEach(() => {
    reactHooks.resetAll();
    ResizeObserverMock.callbacks = [];
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('视角在底部时，新消息会自动滚动到底部', () => {
    const area = createScrollArea();
    const bottom = { scrollIntoView: vi.fn() };
    const first = renderUseChatScroll({ conversationID: 1, messagesLength: 1, latestMessageKey: '1-a-1-100' });
    first.areaRef.current = area;
    first.contentRef.current = {};
    first.bottomRef.current = bottom;
    flushLayoutEffects();

    area.scrollTop = 700;
    first.result.handleAreaScroll();
    area.scrollHeight = 1200;
    const second = renderUseChatScroll({ conversationID: 1, messagesLength: 2, latestMessageKey: '2-b-2-101' });
    flushLayoutEffects();

    expect(second.areaRef.current?.scrollTop).toBe(1200);
    expect(bottom.scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth' });
  });

  it('用户上翻看历史时，新消息不会强制拉回底部', () => {
    const area = createScrollArea({ scrollTop: 700 });
    const bottom = { scrollIntoView: vi.fn() };
    const first = renderUseChatScroll({ conversationID: 1, messagesLength: 1, latestMessageKey: '1-a-1-100' });
    first.areaRef.current = area;
    first.contentRef.current = {};
    first.bottomRef.current = bottom;
    flushLayoutEffects();

    area.scrollTop = 100;
    first.result.handleAreaScroll();
    area.scrollHeight = 1200;
    renderUseChatScroll({ conversationID: 1, messagesLength: 2, latestMessageKey: '2-b-2-101' });
    flushLayoutEffects();

    expect(area.scrollTop).toBe(100);
    expect(bottom.scrollIntoView).not.toHaveBeenCalledWith({ behavior: 'smooth' });
  });

  it('贴底时内容异步撑高会继续保持到底部', () => {
    const area = createScrollArea();
    const first = renderUseChatScroll({ conversationID: 1, messagesLength: 1, latestMessageKey: '1-a-1-100' });
    first.areaRef.current = area;
    first.contentRef.current = {};
    first.bottomRef.current = { scrollIntoView: vi.fn() };
    flushLayoutEffects();

    area.scrollTop = 700;
    first.result.handleAreaScroll();
    area.scrollHeight = 1400;
    ResizeObserverMock.callbacks.forEach((callback) => callback([], {} as ResizeObserver));

    expect(area.scrollTop).toBe(1400);
  });
});
