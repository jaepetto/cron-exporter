export interface EmbeddedConfig {
  basePath: string;
  title: string;
}

function readMeta(name: string): string | null {
  return document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)?.content ?? null;
}

function usableValue(value: string | null, fallback: string): string {
  return value && !value.startsWith('__CRONMETRICS_') ? value : fallback;
}

export const embeddedConfig: EmbeddedConfig = {
  basePath: usableValue(readMeta('cronmetrics-base-path'), '/dashboard').replace(/\/$/, ''),
  title: usableValue(readMeta('cronmetrics-title'), 'Cron Monitor'),
};