import type { Job } from '../api/client';

export function isJobOverdue(job: Job, now = Date.now()): boolean {
  if (job.status !== 'active') return false;
  const reportedAt = Date.parse(job.last_reported_at);
  return Number.isFinite(reportedAt) && now - reportedAt > job.automatic_failure_threshold * 1000;
}

export function formatRelativeTime(timestamp: string, now = Date.now()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now - Date.parse(timestamp)) / 1000));
  if (elapsedSeconds < 60) return 'just now';
  const minutes = Math.floor(elapsedSeconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function formatThreshold(seconds: number): string {
  if (seconds % 86400 === 0) return `${seconds / 86400}d`;
  if (seconds % 3600 === 0) return `${seconds / 3600}h`;
  if (seconds % 60 === 0) return `${seconds / 60}m`;
  return `${seconds}s`;
}