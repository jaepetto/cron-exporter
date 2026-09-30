import { useEffect, useState } from 'react';
import { useSWRConfig } from 'swr';

import type { DashboardEvent } from '../api/client';
import { embeddedConfig } from '../lib/runtime';
import { useRuntimeConfig } from './useJobs';

const invalidationEvents = ['job-status-change', 'job-created', 'job-updated', 'job-deleted', 'reset'] as const;

function isJobCacheKey(key: unknown) {
  return Array.isArray(key) && (key[0] === 'jobs' || key[0] === 'job');
}

export function useDashboardEvents() {
  const { mutate } = useSWRConfig();
  const runtime = useRuntimeConfig();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!runtime.data?.sse_enabled) return;

    const source = new EventSource(`${embeddedConfig.basePath}/events`);
    const invalidate = (event: Event) => {
      const message = event as MessageEvent<string>;
      let parsed: DashboardEvent | undefined;
      try {
        parsed = JSON.parse(message.data) as DashboardEvent;
      } catch {
        parsed = undefined;
      }

      void mutate((key) => {
        if (!Array.isArray(key)) return false;
        if (key[0] === 'jobs') return true;
        return key[0] === 'job' && (parsed?.job_id === undefined || key[1] === parsed.job_id);
      });
    };
    const markConnected = () => setConnected(true);
    const markDisconnected = () => setConnected(false);
    const handleOpen = () => {
      markConnected();
      void mutate(isJobCacheKey);
    };

    source.addEventListener('connected', markConnected);
    source.addEventListener('open', handleOpen);
    source.addEventListener('error', markDisconnected);
    invalidationEvents.forEach((eventType) => source.addEventListener(eventType, invalidate));

    return () => {
      source.removeEventListener('connected', markConnected);
      source.removeEventListener('open', handleOpen);
      source.removeEventListener('error', markDisconnected);
      invalidationEvents.forEach((eventType) => source.removeEventListener(eventType, invalidate));
      source.close();
    };
  }, [mutate, runtime.data?.sse_enabled]);

  return { connected, enabled: runtime.data?.sse_enabled ?? false };
}
