import { Run, RunStatus } from '@/types';

interface RunProgressProps {
  run: Run;
  showDetails?: boolean;
}

export default function RunProgress({ run, showDetails = false }: RunProgressProps) {
  const getStatusColor = (status: RunStatus) => {
    switch (status) {
      case 'completed':
        return 'bg-green-500';
      case 'running':
        return 'bg-blue-500';
      case 'failed':
        return 'bg-red-500';
      case 'cancelled':
        return 'bg-gray-500';
      case 'queued':
        return 'bg-yellow-500';
      case 'retrying':
        return 'bg-orange-500';
      default:
        return 'bg-gray-300';
    }
  };

  const getStatusText = (status: RunStatus) => {
    switch (status) {
      case 'pending':
        return 'Pending';
      case 'queued':
        return 'Queued';
      case 'running':
        return 'Running';
      case 'completed':
        return 'Completed';
      case 'failed':
        return 'Failed';
      case 'cancelled':
        return 'Cancelled';
      case 'retrying':
        return 'Retrying';
      default:
        return status;
    }
  };

  const isActive = run.status === 'running' || run.status === 'queued';
  const showProgress = isActive && run.progress > 0;

  return (
    <div className="space-y-2">
      {/* Status and Progress Bar */}
      <div className="flex items-center space-x-3">
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium text-white ${getStatusColor(
            run.status
          )}`}
        >
          {getStatusText(run.status)}
        </span>
        
        {showProgress && (
          <div className="flex-1 min-w-0">
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${Math.min(100, Math.max(0, run.progress))}%` }}
              />
            </div>
            <div className="text-xs text-gray-500 mt-1">
              {run.progress.toFixed(1)}%
            </div>
          </div>
        )}
      </div>

      {/* Status Message */}
      {showDetails && run.status_message && (
        <div className="text-sm text-gray-600">
          {run.status_message}
        </div>
      )}

      {/* Error Message */}
      {showDetails && run.status === 'failed' && run.last_error && (
        <div className="text-sm text-red-600 bg-red-50 p-2 rounded">
          <strong>Error:</strong> {run.last_error}
        </div>
      )}

      {/* Retry Information */}
      {showDetails && run.retry_count && run.retry_count > 0 && (
        <div className="text-xs text-gray-500">
          Retry attempt: {run.retry_count}
        </div>
      )}

      {/* Worker Information */}
      {showDetails && run.worker_id && run.status === 'running' && (
        <div className="text-xs text-gray-500">
          Worker: {run.worker_id}
        </div>
      )}
    </div>
  );
}