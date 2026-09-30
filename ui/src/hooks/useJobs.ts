import useSWR from 'swr';

import { getJob, getJobs, getRuntimeConfig, type JobsQuery } from '../api/client';

export function useRuntimeConfig() {
  return useSWR('dashboard-config', getRuntimeConfig, { revalidateOnFocus: false });
}

export function useJobs(query: JobsQuery) {
  const runtime = useRuntimeConfig();
  const refreshInterval = runtime.data?.polling_fallback ? runtime.data.polling_interval * 1000 : 0;
  return useSWR(['jobs', query], () => getJobs(query), { keepPreviousData: true, refreshInterval });
}

export function useJob(id: number | null) {
  const runtime = useRuntimeConfig();
  const refreshInterval = runtime.data?.polling_fallback ? runtime.data.polling_interval * 1000 : 0;
  return useSWR(id ? ['job', id] : null, () => getJob(id as number), { refreshInterval });
}
