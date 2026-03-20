import { useState, useEffect, useRef } from 'react';

interface LogEntry {
  direction: string;
  line: string;
  timestamp?: string;
}

interface SendResultItem {
  index: number;
  success: boolean;
  error?: string;
  from?: string;
  to?: string;
  subject?: string;
  timestamp?: string;
}

interface ProgressEvent {
  sent: number;
  failed: number;
  total: number;
}

interface Props {
  logs: LogEntry[];
  results: SendResultItem[];
  progress: ProgressEvent;
  sending: boolean;
  onClearLog: () => void;
  onClearResults: () => void;
}

export default function BottomPanel({ logs, results, progress, sending, onClearLog, onClearResults }: Props) {
  const [activeTab, setActiveTab] = useState<'log' | 'results'>('results');
  const [panelHeight, setPanelHeight] = useState(180);
  const logRef = useRef<HTMLDivElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);
  const startY = useRef(0);
  const startHeight = useRef(0);

  // Auto-scroll log to bottom
  useEffect(() => {
    if (logRef.current && activeTab === 'log') {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logs, activeTab]);

  // Auto-scroll results to top (newest first since reversed)
  useEffect(() => {
    if (resultsRef.current && activeTab === 'results') {
      resultsRef.current.scrollTop = 0;
    }
  }, [results, activeTab]);

  // Resizer handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    isDragging.current = true;
    startY.current = e.clientY;
    startHeight.current = panelHeight;
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  const handleMouseMove = (e: MouseEvent) => {
    if (!isDragging.current) return;
    const delta = startY.current - e.clientY;
    const newHeight = Math.max(80, Math.min(600, startHeight.current + delta));
    setPanelHeight(newHeight);
  };

  const handleMouseUp = () => {
    isDragging.current = false;
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  };

  const pct = progress.total > 0
    ? ((progress.sent + progress.failed) / progress.total) * 100
    : 0;

  const statusText = sending
    ? `Sending... ${progress.sent + progress.failed}/${progress.total}`
    : progress.total > 0
      ? `Done — ${progress.sent} sent, ${progress.failed} failed`
      : 'Ready';

  return (
    <div className="bottom-panel" style={{ height: panelHeight }}>
      <div className="bottom-panel-resizer" onMouseDown={handleMouseDown} />

      <div className="bottom-tabs">
        <div
          className={`bottom-tab ${activeTab === 'results' ? 'active' : ''}`}
          onClick={() => setActiveTab('results')}
        >
          Results {results.length > 0 ? `(${results.length})` : ''}
        </div>
        <div
          className={`bottom-tab ${activeTab === 'log' ? 'active' : ''}`}
          onClick={() => setActiveTab('log')}
        >
          SMTP Log
        </div>
        <div className="bottom-spacer" />
        {activeTab === 'log' && (
          <button className="bottom-clear-btn" onClick={onClearLog}>
            Clear
          </button>
        )}
        {activeTab === 'results' && (
          <button className="bottom-clear-btn" onClick={onClearResults}>
            Clear
          </button>
        )}
      </div>

      <div className="bottom-content">
        {activeTab === 'log' ? (
          <div className="log-output" ref={logRef}>
            {logs.map((l, i) => (
              <div
                key={i}
                className={`log-line ${l.direction === 'client' ? 'log-client' : 'log-server'}`}
              >
                {l.timestamp && <span className="log-ts">{l.timestamp}</span>}
                {l.direction === 'client' ? ' → ' : ' ← '}{l.line}
              </div>
            ))}
          </div>
        ) : (
          <div className="results-output" ref={resultsRef}>
            {results.slice(-200).reverse().map((r, i) => (
              <div
                className={`result-item ${r.success ? 'success' : 'failed'}`}
                key={`${r.index}-${i}`}
              >
                {r.timestamp && <span className="result-ts">{r.timestamp}</span>}
                {r.index >= 0 ? (
                  <>
                    <span className="result-index">#{r.index}</span>
                    <span className="result-envelope">[{r.from} &gt; {r.to}]</span>
                    <span className="result-subject">[{r.subject}]</span>
                    <span className="result-status">{r.success ? 'OK' : r.error}</span>
                  </>
                ) : (
                  <span className="result-system">{r.subject}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="progress-strip">
        <div className="mini-progress">
          <div
            className={`mini-progress-fill ${progress.failed > 0 ? 'has-errors' : ''}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        <span className="stat">Sent: <b>{progress.sent}</b></span>
        <span className="stat">Failed: <b>{progress.failed}</b></span>
        <span className="stat">Total: <b>{progress.total}</b></span>
        <div style={{ flex: 1 }} />
        <span className="stat">{statusText}</span>
      </div>
    </div>
  );
}
