import { ArrowLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return <div className="empty-state full-page"><span className="error-code">404</span><h1>Page not found</h1><Link className="button button-primary" to="/jobs"><ArrowLeft size={16} />Back to jobs</Link></div>;
}
