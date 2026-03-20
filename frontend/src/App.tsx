import { useState, useEffect, useCallback } from 'react';
import { core } from '../wailsjs/go/models';
import {
  LoadConfig,
  SaveConfig,
  SyncConfig,
  SavePassword,
  LoadPassword,
  StartSend,
  StopSend,
  ClearLog,
  SaveWindowState,
  SelectEMLFiles,
  GetConfigPath,
  LoadConfigFrom,
  StartSendEML,
  GetVersion,
  CheckVersion,
  ParseEMLPreview,
  SelectAttachments,
  RemoveAttachment,
  ClearAttachments,
} from '../wailsjs/go/main/AppService';
import { EventsOn } from '../wailsjs/runtime/runtime';

import SettingsPanel from './components/SettingsPanel';
import EditorHeader from './components/EditorHeader';
import BottomPanel from './components/BottomPanel';

interface ProgressEvent {
  sent: number;
  failed: number;
  total: number;
}

interface SendResultItem {
  index: number;
  success: boolean;
  error?: string;
  from?: string;
  to?: string;
  subject?: string;
}

interface LogEntry {
  direction: string;
  line: string;
}

function App() {
  const [settingsOpen, setSettingsOpen] = useState(true);
  const [sendMode, setSendMode] = useState<'input' | 'eml'>('input');
  const [emlFiles, setEmlFiles] = useState<string[]>([]);
  const [emlPreview, setEmlPreview] = useState<{ subject: string; from: string; to: string; body: string; content_type: string } | null>(null);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [attachments, setAttachments] = useState<string[]>([]);
  const [statusMessage, setStatusMessage] = useState('');
  const [saveMessage, setSaveMessage] = useState('');
  const [configPath, setConfigPath] = useState('');
  const [appVersion, setAppVersion] = useState('');
  const [config, setConfig] = useState<core.AppConfig>(
    new core.AppConfig({
      server: { smtp: 'localhost', port: 25, tls: false, ssl: false, tls_version: '1.3', skip_verify: true, auth: false, auth_id: '' },
      mail: {
        mail_from: '', numbering_mail_from: false,
        rcpt_to: '', numbering_rcpt_to: false,
        subject: '', numbering_subject: false, timestamp_subject: false,
        body: '', content_type: 'text/plain',
        mail_number: 1, thread_number: 1, interval_ms: 0,
        use_header_envelope: false, update_message_id: false,
        custom_headers: [],
      },
      window: { x: 0, y: 0, width: 1024, height: 720 },
      theme: 'dark',
    })
  );
  const [password, setPassword] = useState('');
  const [sending, setSending] = useState(false);
  const [progress, setProgress] = useState<ProgressEvent>({ sent: 0, failed: 0, total: 0 });
  const [results, setResults] = useState<SendResultItem[]>([]);
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const isEml = sendMode === 'eml';

  // Apply theme to document
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  // Load config on startup
  useEffect(() => {
    LoadConfig().then((cfg) => {
      setConfig(cfg);
      if (cfg.theme === 'light' || cfg.theme === 'dark') {
        setTheme(cfg.theme);
      }
      if (cfg.server.auth && cfg.server.auth_id) {
        LoadPassword(cfg.server.auth_id).then(setPassword).catch(() => {});
      }
    });
    GetConfigPath().then(setConfigPath).catch(() => {});
    GetVersion().then(setAppVersion).catch(() => {});

    // Version check on startup
    CheckVersion().then((v: { current: string; latest: string; update_avail: boolean; download_url?: string; error?: string }) => {
      if (v.error) {
        setResults((prev) => [...prev, {
          index: -1, success: false,
          from: '', to: '', subject: `[Version] ${v.error}`,
        }]);
      } else if (v.update_avail) {
        setResults((prev) => [...prev, {
          index: -1, success: true,
          from: '', to: '', subject: `[Update] New version available: ${v.latest} (current: ${v.current})`,
        }]);
      } else {
        setResults((prev) => [...prev, {
          index: -1, success: true,
          from: '', to: '', subject: `[Version] Up to date (${v.current})`,
        }]);
      }
    }).catch(() => {});
  }, []);

  // Listen for Wails events
  useEffect(() => {
    const cleanups = [
      EventsOn('progress', (p: ProgressEvent) => setProgress(p)),
      EventsOn('result', (r: SendResultItem) =>
        setResults((prev) => [...prev, r])
      ),
      EventsOn('done', () => setSending(false)),
      EventsOn('smtp:log', (entry: LogEntry) =>
        setLogs((prev) => [...prev, entry])
      ),
      EventsOn('log:clear', () => setLogs([])),
      EventsOn('config:saved', (data: { path: string }) => {
        setConfigPath(data.path);
        showSaveMsg(`Saved`);
        setResults((prev) => [...prev, {
          index: -1, success: true,
          from: '', to: '', subject: `[Config] Saved to ${data.path}`,
        }]);
      }),
      EventsOn('config:loaded', (data: { path: string }) => {
        setConfigPath(data.path);
        showSaveMsg(`Loaded`);
        setResults((prev) => [...prev, {
          index: -1, success: true,
          from: '', to: '', subject: `[Config] Loaded from ${data.path}`,
        }]);
      }),
      EventsOn('test:result', (data: { success: boolean; error?: string; server: string }) => {
        setResults((prev) => [...prev, {
          index: -1,
          success: data.success,
          from: '', to: data.server,
          subject: data.success
            ? `[Test] Connection OK`
            : `[Test] Failed: ${data.error}`,
        }]);
      }),
    ];
    return () => cleanups.forEach((fn) => fn());
  }, []);

  // Show a temporary status message
  const showStatus = (msg: string) => {
    setStatusMessage(msg);
    setTimeout(() => setStatusMessage(''), 4000);
  };

  // Show save/load feedback next to Save button
  const showSaveMsg = (msg: string) => {
    setSaveMessage(msg);
    setTimeout(() => setSaveMessage(''), 4000);
  };

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey && e.key === 'Enter') || e.key === 'F5') {
        e.preventDefault();
        if (!sending) handleStartSend();
      } else if (e.ctrlKey && e.key === 's') {
        e.preventDefault();
        handleSave();
      } else if (e.ctrlKey && e.key === 'l') {
        e.preventDefault();
        handleClearLog();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [config, sending, password]);

  // Save window state on resize
  useEffect(() => {
    let timeout: ReturnType<typeof setTimeout>;
    const handler = () => {
      clearTimeout(timeout);
      timeout = setTimeout(() => {
        SaveWindowState({
          x: window.screenX,
          y: window.screenY,
          width: window.outerWidth,
          height: window.outerHeight,
        } as core.WindowState);
      }, 500);
    };
    window.addEventListener('resize', handler);
    return () => {
      window.removeEventListener('resize', handler);
      clearTimeout(timeout);
    };
  }, []);

  const handleSave = useCallback(async () => {
    try {
      await SaveConfig(config);
      if (config.server.auth && config.server.auth_id && password) {
        await SavePassword(config.server.auth_id, password);
      }
    } catch (e) {
      showStatus('Save failed: ' + e);
    }
  }, [config, password]);

  const handleLoadConfig = useCallback(async () => {
    try {
      const cfg = await LoadConfigFrom();
      setConfig(cfg);
      if (cfg.server.auth && cfg.server.auth_id) {
        LoadPassword(cfg.server.auth_id).then(setPassword).catch(() => {});
      }
    } catch (e) {
      showStatus('Load failed: ' + e);
    }
  }, []);

  const handleStartSend = useCallback(async () => {
    try {
      // --- Input validation (required fields + range checks only) ---
      const errors: string[] = [];

      // Server validation
      if (!config.server.smtp.trim()) errors.push('SMTP host is required');
      if (config.server.port < 1 || config.server.port > 65535) errors.push('Port must be 1–65535');
      if (config.server.auth) {
        if (!config.server.auth_id.trim()) errors.push('Auth ID is required when AUTH is enabled');
        if (!password.trim()) errors.push('Password is required when AUTH is enabled');
      }

      // From/To required check (except EML mode + UseHeaderEnvelope=ON)
      const needFromTo = !(isEml && config.mail.use_header_envelope);
      if (needFromTo) {
        if (!config.mail.mail_from.trim()) errors.push('From address is required');
        if (!config.mail.rcpt_to.trim()) errors.push('To address is required');
      }

      // Send option range checks
      if (config.mail.mail_number < 1 || config.mail.mail_number > 100000) errors.push('Count must be 1–100,000');
      if (config.mail.thread_number < 1) errors.push('Threads must be at least 1');
      if (config.mail.interval_ms < 0 || config.mail.interval_ms > 60000) errors.push('Interval must be 0–60,000 ms');

      if (errors.length > 0) {
        showStatus(errors[0]);
        errors.forEach((msg) => {
          setResults((prev) => [...prev, { index: -1, success: false, from: '', to: '', subject: `[Validation] ${msg}` }]);
        });
        return;
      }

      // Sync current frontend config to backend (in-memory only, no disk write)
      await SyncConfig(config);

      // --- Collect warnings for confirmation dialog ---
      const warnings: string[] = [];
      const sendCount = isEml ? (config.mail.mail_number || emlFiles.length) : config.mail.mail_number;

      if (isEml && emlFiles.length === 0) {
        showStatus('No EML files selected');
        return;
      }

      if (sendCount >= 100) {
        warnings.push(`${sendCount.toLocaleString()}건의 메일을 전송합니다.`);
      }
      if (config.mail.thread_number > 50) {
        warnings.push(`스레드 수가 ${config.mail.thread_number}개로 설정되어 있습니다. 서버에 부하가 발생할 수 있습니다.`);
      }
      if (config.server.auth && !config.server.tls && !config.server.ssl) {
        warnings.push(`TLS/SSL 없이 AUTH를 사용합니다. 자격 증명이 평문으로 전송됩니다.`);
      }

      if (warnings.length > 0) {
        const msg = warnings.join('\n') + '\n\n진행하시겠습니까?';
        if (!confirm(msg)) return;
      }

      if (isEml) {
        const emlCount = config.mail.mail_number || emlFiles.length;
        setProgress({ sent: 0, failed: 0, total: emlCount });
        setSending(true);
        await StartSendEML(
          emlFiles,
          config.mail.mail_from,
          config.mail.rcpt_to,
          config.mail.numbering_mail_from,
          config.mail.numbering_rcpt_to,
          config.mail.use_header_envelope,
          config.mail.update_message_id,
          emlCount,
          config.mail.thread_number,
          config.mail.interval_ms,
        );
      } else {
        setProgress({ sent: 0, failed: 0, total: config.mail.mail_number });
        setSending(true);
        await StartSend();
      }
    } catch (e) {
      setSending(false);
      showStatus('Send error: ' + e);
    }
  }, [config, password, isEml, emlFiles]);

  const handleStopSend = useCallback(() => {
    StopSend();
  }, []);

  const handleClearLog = useCallback(() => {
    ClearLog();
    setLogs([]);
  }, []);

  const handleClearResults = useCallback(() => {
    setResults([]);
    setProgress({ sent: 0, failed: 0, total: 0 });
  }, []);

  const handleSelectEML = async () => {
    try {
      const files = await SelectEMLFiles();
      if (files && files.length > 0) {
        setEmlFiles(files);
        // Load preview for the first file
        try {
          const preview = await ParseEMLPreview(files[0]);
          setEmlPreview(preview);
        } catch {
          setEmlPreview(null);
        }
      }
    } catch (e) {
      console.error('Failed to select EML files:', e);
    }
  };

  const handleClearEML = () => {
    setEmlFiles([]);
    setEmlPreview(null);
  };

  const handleAddAttachment = async () => {
    try {
      const files = await SelectAttachments();
      if (files) {
        setAttachments(files);
      }
    } catch (e) {
      console.error('Failed to select attachments:', e);
    }
  };

  const removeAttachment = async (index: number) => {
    try {
      const files = await RemoveAttachment(index);
      setAttachments(files || []);
    } catch (e) {
      console.error('Failed to remove attachment:', e);
    }
  };

  const handleClearAttachments = async () => {
    await ClearAttachments();
    setAttachments([]);
  };

  // Status bar info
  const tlsInfo = config.server.ssl
    ? `SSL/TLS ${config.server.tls_version || '1.3'}`
    : config.server.tls
      ? `STARTTLS ${config.server.tls_version || '1.3'}`
      : 'No TLS';
  const serverInfo = `${config.server.smtp}:${config.server.port}`;
  const authInfo = config.server.auth ? `AUTH: ${config.server.auth_id}` : '';

  return (
    <div className="app">
      {/* Toolbar */}
      <div className="toolbar">
        <span className="toolbar-logo">dMailSender{appVersion && <span className="toolbar-version">v{appVersion}</span>}</span>
        <span className="toolbar-sep" />
        <div className="toolbar-group">
          <button
            className="tool-btn send"
            onClick={handleStartSend}
            disabled={sending}
          >
            Send (F5)
          </button>
          <button
            className="tool-btn stop"
            onClick={handleStopSend}
            disabled={!sending}
          >
            Stop
          </button>
        </div>
        <span className="toolbar-sep" />
        <div className="toolbar-group">
          <button className="tool-btn" onClick={handleSave}>Save</button>
          <button className="tool-btn" onClick={handleLoadConfig}>Load</button>
          {saveMessage && <span className="save-feedback">{saveMessage}</span>}
        </div>
        <div className="toolbar-spacer" />
        <div className="toolbar-group">
          <select
            value={sendMode}
            onChange={(e) => setSendMode(e.target.value as 'input' | 'eml')}
          >
            <option value="input">Input Mode</option>
            <option value="eml">EML Mode</option>
          </select>
        </div>
        <span className="toolbar-sep" />
        <div className="toolbar-group">
          <button
            className="tool-btn theme-toggle"
            onClick={() => {
              const newTheme = theme === 'dark' ? 'light' : 'dark';
              setTheme(newTheme);
              setConfig((prev) => new core.AppConfig({ ...prev, theme: newTheme }));
            }}
            title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
          >
            {theme === 'dark' ? '☀' : '☾'}
          </button>
        </div>
      </div>

      {/* Main */}
      <div className="main">
        {/* Settings panel (collapsible) */}
        {settingsOpen ? (
          <SettingsPanel
            config={config}
            password={password}
            sendMode={sendMode}
            onChange={setConfig}
            onPasswordChange={setPassword}
            onClose={() => setSettingsOpen(false)}
          />
        ) : (
          <button
            className="settings-open-btn"
            onClick={() => setSettingsOpen(true)}
            title="Open Settings"
          >
            &#x25B8;
          </button>
        )}

        {/* Content area */}
        <div className="content-area">
          {/* Envelope (From/To) — always visible */}
          <EditorHeader config={config} onChange={setConfig} envelopeDisabled={false} contentDisabled={isEml} />

          {isEml ? (
            /* EML Mode: file selector + preview + options */
            <div className="eml-panel">
              <div className="eml-toolbar">
                <button className="tool-btn" onClick={handleSelectEML}>
                  Select EML File
                </button>
                {emlFiles.length > 0 && (
                  <span className="eml-file-chip">
                    <span className="eml-file-path" title={emlFiles[0]}>{emlFiles[0].split(/[\\/]/).pop()}</span>
                    <button className="btn-remove" onClick={handleClearEML}>×</button>
                  </span>
                )}
              </div>

              <div className="eml-preview">
                {emlPreview ? (
                  <>
                    <div className="eml-preview-header">
                      <div className="eml-preview-field">
                        <span className="eml-preview-label">Subject</span>
                        <span className="eml-preview-value">{emlPreview.subject || '(no subject)'}</span>
                      </div>
                      {emlPreview.from && (
                        <div className="eml-preview-field">
                          <span className="eml-preview-label">From</span>
                          <span className="eml-preview-value">{emlPreview.from}</span>
                        </div>
                      )}
                      {emlPreview.to && (
                        <div className="eml-preview-field">
                          <span className="eml-preview-label">To</span>
                          <span className="eml-preview-value">{emlPreview.to}</span>
                        </div>
                      )}
                    </div>
                    <div className="eml-preview-body">
                      <textarea readOnly value={emlPreview.body || '(empty body)'} />
                    </div>
                  </>
                ) : (
                  <div className="eml-preview-empty">
                    Select an EML file to preview its contents
                  </div>
                )}
              </div>

              <div className="eml-advanced">
                <div className="eml-advanced-title">EML Options</div>
                <div className="s-toggle">
                  <span>Use Header Envelope</span>
                  <label className="toggle">
                    <input
                      type="checkbox"
                      checked={config.mail.use_header_envelope}
                      onChange={(e) => {
                        if (e.target.checked) {
                          if (!window.confirm('헤더 발신자/수신자 정보를 사용하여 메일을 전송합니다.\n의도하지 않은 주소로 발송될 수 있으니 주의가 필요합니다.\n\n진행하시겠습니까?')) {
                            return;
                          }
                        }
                        setConfig(new core.AppConfig({
                          ...config,
                          mail: new core.MailConfig({ ...config.mail, use_header_envelope: e.target.checked }),
                        }));
                      }}
                    />
                    <span className="slider" />
                  </label>
                </div>
                <div className="s-toggle">
                  <span>Update Message-ID</span>
                  <label className="toggle">
                    <input
                      type="checkbox"
                      checked={config.mail.update_message_id}
                      onChange={(e) =>
                        setConfig(new core.AppConfig({
                          ...config,
                          mail: new core.MailConfig({ ...config.mail, update_message_id: e.target.checked }),
                        }))
                      }
                    />
                    <span className="slider" />
                  </label>
                </div>
              </div>
            </div>
          ) : (
            /* Input Mode: attachment bar + body editor */
            <>
              <div className="attachment-bar">
                <button className="attach-btn" onClick={handleAddAttachment}>
                  Attach
                </button>
                {attachments.map((f, i) => (
                  <span className="attach-chip" key={i}>
                    {f.split(/[\\/]/).pop()}
                    <button className="attach-remove" onClick={() => removeAttachment(i)}>×</button>
                  </span>
                ))}
              </div>

              <div className="editor-body">
                <textarea
                  value={config.mail.body}
                  onChange={(e) =>
                    setConfig(new core.AppConfig({
                      ...config,
                      mail: new core.MailConfig({ ...config.mail, body: e.target.value }),
                    }))
                  }
                  placeholder={
                    config.mail.content_type === 'text/html'
                      ? '<html><body>Your HTML content here</body></html>'
                      : 'Your message here...'
                  }
                />
              </div>
            </>
          )}

          <BottomPanel
            logs={logs}
            results={results}
            progress={progress}
            sending={sending}
            onClearLog={handleClearLog}
            onClearResults={handleClearResults}
          />
        </div>
      </div>

      {/* Status bar */}
      <div className="status-bar">
        <span>
          {statusMessage
            ? statusMessage
            : `${serverInfo} | ${tlsInfo}${authInfo ? ` | ${authInfo}` : ''}`
          }
        </span>
        <span className="status-center" title={configPath ? `Config: ${configPath}` : ''}>
          Ctrl+Enter: Send | Ctrl+S: Save | Ctrl+L: Clear Log
        </span>
        <span className="status-copyright">Created by aimaya</span>
      </div>
    </div>
  );
}

export default App;
