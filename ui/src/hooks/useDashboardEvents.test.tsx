import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useDashboardEvents } from './useDashboardEvents';

const mocks = vi.hoisted(() => ({
  mutate: vi.fn(),
  runtimeConfig: { data: { sse_enabled: true } },
}));

vi.mock('swr', () => ({ useSWRConfig: () => ({ mutate: mocks.mutate }) }));
vi.mock('./useJobs', () => ({ useRuntimeConfig: () => mocks.runtimeConfig }));

class MockEventSource {
  static latest: MockEventSource | undefined;
  private listeners = new Map<string, Set<EventListenerOrEventListenerObject>>();

  constructor() {
    MockEventSource.latest = this;
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const listeners = this.listeners.get(type) ?? new Set();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    this.listeners.get(type)?.delete(listener);
  }

  close() {}

  emit(type: string) {
    const event = new Event(type);
    this.listeners.get(type)?.forEach((listener) => {
      if (typeof listener === 'function') listener.call(this, event);
      else listener.handleEvent(event);
    });
  }
}

describe('useDashboardEvents', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    mocks.mutate.mockReset();
    MockEventSource.latest = undefined;
  });

  it('revalidates job caches after EventSource reconnects', () => {
    vi.stubGlobal('EventSource', MockEventSource);
    const { result } = renderHook(() => useDashboardEvents());
    const source = MockEventSource.latest;
    expect(source).toBeDefined();

    act(() => source?.emit('open'));
    act(() => source?.emit('error'));
    mocks.mutate.mockClear();
    act(() => source?.emit('open'));

    expect(result.current.connected).toBe(true);
    expect(mocks.mutate).toHaveBeenCalledTimes(1);
    const filter = mocks.mutate.mock.calls[0][0] as (key: unknown) => boolean;
    expect(filter(['jobs', { page: 1 }])).toBe(true);
    expect(filter(['job', 7])).toBe(true);
    expect(filter('dashboard-config')).toBe(false);
  });
});
