import { ArrowLeft, Save } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';

import { APIError, createJob, updateJob, type CreateJobInput, type Job } from '../api/client';
import { Button, ErrorState, Field, LoadingState, SelectField } from '../components/ui';
import { useJob } from '../hooks/useJobs';

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'maintenance', label: 'Maintenance' },
  { value: 'paused', label: 'Paused' },
];

interface FormState {
  job_name: string;
  host: string;
  automatic_failure_threshold: string;
  status: 'active' | 'maintenance' | 'paused';
  labels: string;
}

const emptyForm: FormState = {
  job_name: '',
  host: '',
  automatic_failure_threshold: '3600',
  status: 'active',
  labels: '{}',
};

export function JobFormPage({ mode }: { mode: 'create' | 'edit' }) {
  const routeID = Number(useParams().id);
  const id = mode === 'edit' && Number.isInteger(routeID) && routeID > 0 ? routeID : null;
  const existing = useJob(id);

  if (mode === 'edit' && id === null) return <ErrorState message="Invalid job ID" />;
  if (mode === 'edit' && existing.isLoading) return <LoadingState label="Loading job" />;
  if (mode === 'edit' && existing.error) return <ErrorState message={existing.error instanceof Error ? existing.error.message : 'Unable to load job'} />;
  if (mode === 'edit' && !existing.data) return <ErrorState message="Job not found" />;

  const initialForm = existing.data ? formFromJob(existing.data) : emptyForm;
  return <JobEditor key={existing.data?.updated_at ?? 'create'} mode={mode} id={id} initialForm={initialForm} />;
}

function formFromJob(job: Job): FormState {
  return {
    job_name: job.job_name,
    host: job.host,
    automatic_failure_threshold: String(job.automatic_failure_threshold),
    status: job.status,
    labels: JSON.stringify(job.labels, null, 2),
  };
}

function JobEditor({ mode, id, initialForm }: { mode: 'create' | 'edit'; id: number | null; initialForm: FormState }) {
  const navigate = useNavigate();
  const [form, setForm] = useState<FormState>(initialForm);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setFields({});
    setError(undefined);
    let labels: Record<string, string>;
    try {
      const parsedLabels: unknown = JSON.parse(form.labels);
      if (typeof parsedLabels !== 'object' || parsedLabels === null || Array.isArray(parsedLabels) || Object.values(parsedLabels).some((value) => typeof value !== 'string')) throw new Error();
      labels = parsedLabels as Record<string, string>;
    } catch {
      setFields({ labels: 'Enter a JSON object whose values are strings.' });
      return;
    }

    const input: CreateJobInput = {
      job_name: form.job_name.trim(),
      host: form.host.trim(),
      automatic_failure_threshold: Number(form.automatic_failure_threshold),
      status: form.status,
      labels,
    };
    setSubmitting(true);
    try {
      const saved = mode === 'create' ? await createJob(input) : await updateJob(id as number, input);
      navigate(`/jobs/${saved.id}`, { replace: true });
    } catch (requestError) {
      if (requestError instanceof APIError) setFields(requestError.fields);
      setError(requestError instanceof Error ? requestError.message : 'Unable to save job');
    } finally {
      setSubmitting(false);
    }
  };

  const update = <Key extends keyof FormState>(key: Key, value: FormState[Key]) => setForm((current) => ({ ...current, [key]: value }));

  return (
    <div className="page-stack form-page">
      <Link className="back-link" to={id ? `/jobs/${id}` : '/jobs'}><ArrowLeft size={16} />{id ? 'Job details' : 'Jobs'}</Link>
      <header className="page-header"><div><p className="eyebrow">{mode === 'create' ? 'Registration' : `Job #${id}`}</p><h1>{mode === 'create' ? 'New job' : 'Edit job'}</h1></div></header>
      {error && <ErrorState message={error} />}
      <form className="job-form" onSubmit={(event) => void submit(event)} noValidate>
        <div className="form-grid">
          <Field label="Job name" htmlFor="job-name" error={fields.job_name}><input id="job-name" value={form.job_name} onChange={(event) => update('job_name', event.target.value)} autoComplete="off" required /></Field>
          <Field label="Host" htmlFor="host" error={fields.host}><input id="host" value={form.host} onChange={(event) => update('host', event.target.value)} autoComplete="off" required /></Field>
          <Field label="Failure threshold (seconds)" htmlFor="threshold" error={fields.automatic_failure_threshold}><input id="threshold" type="number" min="1" value={form.automatic_failure_threshold} onChange={(event) => update('automatic_failure_threshold', event.target.value)} required /></Field>
          <Field label="Lifecycle status" htmlFor="job-status" error={fields.status}><SelectField id="job-status" value={form.status} onValueChange={(value) => update('status', value as FormState['status'])} options={statusOptions} placeholder="Status" /></Field>
        </div>
        <Field label="Labels (JSON)" htmlFor="labels" error={fields.labels}><textarea id="labels" rows={8} value={form.labels} onChange={(event) => update('labels', event.target.value)} spellCheck={false} /></Field>
        <div className="form-actions"><Link className="button button-secondary" to={id ? `/jobs/${id}` : '/jobs'}>Cancel</Link><Button type="submit" disabled={submitting}><Save size={16} />{submitting ? 'Saving' : 'Save job'}</Button></div>
      </form>
    </div>
  );
}
