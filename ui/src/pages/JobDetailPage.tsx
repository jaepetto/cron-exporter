import { ArrowLeft, Clock3, Copy, Pencil, Power, Server, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useSWRConfig } from 'swr';

import { deleteJob, toggleJob } from '../api/client';
import { ConfirmDialog, Button, ErrorState, LoadingState, StatusBadge } from '../components/ui';
import { useJob } from '../hooks/useJobs';
import { formatRelativeTime, formatThreshold, isJobOverdue } from '../lib/jobs';

export function JobDetailPage() {
  const id = Number(useParams().id);
  const job = useJob(Number.isInteger(id) && id > 0 ? id : null);
  const { mutate } = useSWRConfig();
  const navigate = useNavigate();
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string>();

  if (!Number.isInteger(id) || id <= 0) return <ErrorState message="Invalid job ID" />;
  if (job.isLoading) return <LoadingState label="Loading job" />;
  if (job.error || !job.data) return <ErrorState message={job.error instanceof Error ? job.error.message : 'Job not found'} retry={() => void job.mutate()} />;

  const toggle = async () => {
    setPending(true);
    setActionError(undefined);
    try {
      await toggleJob(id);
      await Promise.all([job.mutate(), mutate((key) => Array.isArray(key) && key[0] === 'jobs')]);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Unable to update job');
    } finally {
      setPending(false);
    }
  };
  const remove = async () => {
    setPending(true);
    try {
      await deleteJob(id);
      await mutate((key) => Array.isArray(key) && key[0] === 'jobs');
      navigate('/jobs', { replace: true });
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Unable to delete job');
      setPending(false);
    }
  };

  const overdue = isJobOverdue(job.data);
  const apiKey = job.data.api_key;
  return (
    <div className="page-stack detail-page">
      <Link className="back-link" to="/jobs"><ArrowLeft size={16} />Jobs</Link>
      <header className="detail-header">
        <div><p className="eyebrow">Job #{job.data.id}</p><h1>{job.data.job_name}</h1><span className="host"><Server size={16} />{job.data.host}</span></div>
        <div className="header-actions">
          <Link className="button button-secondary" to={`/jobs/${id}/edit`}><Pencil size={16} />Edit</Link>
          <Button variant="secondary" disabled={pending} onClick={() => void toggle()}><Power size={16} />{job.data.status === 'maintenance' ? 'Resume' : 'Maintenance'}</Button>
          <ConfirmDialog trigger={<Button variant="danger" disabled={pending}><Trash2 size={16} />Delete</Button>} title="Delete this job?" description={`${job.data.job_name} on ${job.data.host} will be permanently removed.`} confirmLabel="Delete job" onConfirm={() => void remove()} />
        </div>
      </header>
      {actionError && <ErrorState message={actionError} />}

      <section className="detail-status-band">
        <div><span>Current state</span><StatusBadge status={job.data.status} overdue={overdue} /></div>
        <div><span>Last report</span><strong>{formatRelativeTime(job.data.last_reported_at)}</strong><small>{new Date(job.data.last_reported_at).toLocaleString()}</small></div>
        <div><span>Failure threshold</span><strong>{formatThreshold(job.data.automatic_failure_threshold)}</strong><small><Clock3 size={13} />Since last report</small></div>
      </section>

      <div className="detail-grid">
        <section className="detail-section"><h2>Labels</h2>{Object.keys(job.data.labels).length ? <dl className="label-definitions">{Object.entries(job.data.labels).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl> : <p className="subtle">No labels configured.</p>}</section>
        <section className="detail-section"><h2>Job API key</h2><div className="secret-row"><code>{apiKey ?? 'Not available'}</code>{apiKey && <Button variant="ghost" title="Copy API key" aria-label="Copy API key" onClick={() => void navigator.clipboard.writeText(apiKey)}><Copy size={17} /></Button>}</div><p className="subtle">Use this key only when submitting results for this job.</p></section>
      </div>
    </div>
  );
}