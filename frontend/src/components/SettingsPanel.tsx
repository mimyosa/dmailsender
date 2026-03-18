import { useState, useEffect } from 'react';
import { core } from '../../wailsjs/go/models';
import { TestConnection, SaveConfig } from '../../wailsjs/go/main/AppService';

type SendMode = 'input' | 'eml';

interface Props {
  config: core.AppConfig;
  password: string;
  sendMode: SendMode;
  onChange: (cfg: core.AppConfig) => void;
  onPasswordChange: (pw: string) => void;
  onClose: () => void;
}

export default function SettingsPanel({ config, password, sendMode, onChange, onPasswordChange, onClose }: Props) {
  const [testing, setTesting] = useState(false);
  const server = config.server;
  const mail = config.mail;

  // Local text state for numeric inputs (allows intermediate typing like clearing the field)
  const [countText, setCountText] = useState(String(mail.mail_number));
  const [threadsText, setThreadsText] = useState(String(mail.thread_number));
  const [intervalText, setIntervalText] = useState(String(mail.interval_ms));

  // Sync from external config changes (e.g., config load)
  useEffect(() => setCountText(String(mail.mail_number)), [mail.mail_number]);
  useEffect(() => setThreadsText(String(mail.thread_number)), [mail.thread_number]);
  useEffect(() => setIntervalText(String(mail.interval_ms)), [mail.interval_ms]);

  const updateServer = (patch: Partial<core.ServerConfig>) => {
    onChange(new core.AppConfig({
      ...config,
      server: new core.ServerConfig({ ...server, ...patch }),
    }));
  };

  const updateMail = (patch: Partial<core.MailConfig>) => {
    onChange(new core.AppConfig({
      ...config,
      mail: new core.MailConfig({ ...mail, ...patch }),
    }));
  };

  const updateHeader = (index: number, field: 'key' | 'value', val: string) => {
    const headers = [...(mail.custom_headers || [])];
    headers[index] = { ...headers[index], [field]: val } as core.Header;
    updateMail({ custom_headers: headers });
  };

  const addHeader = () => {
    const headers = [...(mail.custom_headers || []), { key: '', value: '' } as core.Header];
    updateMail({ custom_headers: headers });
  };

  const removeHeader = (index: number) => {
    const headers = (mail.custom_headers || []).filter((_, i) => i !== index);
    updateMail({ custom_headers: headers });
  };

  return (
    <div className="settings-panel">
      <div className="settings-header">
        <span>Settings</span>
        <button className="settings-close-btn" onClick={onClose} title="Close Settings">&#x25C2;</button>
      </div>

      {/* Connection */}
      <div className="s-section">
        <div className="s-title">Connection</div>
        <div className="s-field">
          <label>SMTP Host</label>
          <div className="s-input-with-btn">
            <input
              type="text"
              value={server.smtp}
              onChange={(e) => updateServer({ smtp: e.target.value })}
              placeholder="smtp.example.com"
            />
            <button
              className="btn-test"
              disabled={testing || !server.smtp}
              onClick={async () => {
                setTesting(true);
                try {
                  await SaveConfig(config);
                  await TestConnection();
                } catch (e) {
                  // Error will appear in SMTP log
                } finally {
                  setTesting(false);
                }
              }}
            >
              {testing ? 'Testing...' : 'Test'}
            </button>
          </div>
        </div>
        <div className="s-row">
          <div className="s-field">
            <label>Port</label>
            <input
              type="text"
              inputMode="numeric"
              value={server.port}
              onChange={(e) => {
                const v = parseInt(e.target.value);
                if (!isNaN(v) && v >= 0 && v <= 65535) updateServer({ port: v });
                else if (e.target.value === '') updateServer({ port: 0 });
              }}
            />
          </div>
          <div className="s-field">
            <label>TLS Version</label>
            <select
              value={server.tls_version || '1.3'}
              onChange={(e) => updateServer({ tls_version: e.target.value })}
            >
              <option value="1.3">TLS 1.3</option>
              <option value="1.2">TLS 1.2</option>
              <option value="1.1">TLS 1.1</option>
              <option value="1.0">TLS 1.0</option>
            </select>
          </div>
        </div>
        <div className="s-radio-group">
          <label className="s-radio">
            <input
              type="radio"
              name="tls-mode"
              checked={!server.tls && !server.ssl}
              onChange={() => updateServer({ tls: false, ssl: false })}
            />
            <span>No TLS</span>
          </label>
          <label className="s-radio">
            <input
              type="radio"
              name="tls-mode"
              checked={server.tls && !server.ssl}
              onChange={() => updateServer({ tls: true, ssl: false })}
            />
            <span>STARTTLS</span>
          </label>
          <label className="s-radio">
            <input
              type="radio"
              name="tls-mode"
              checked={server.ssl && !server.tls}
              onChange={() => updateServer({ ssl: true, tls: false, port: 465 })}
            />
            <span>Implicit SSL</span>
          </label>
        </div>
        {(server.tls || server.ssl) && (
          <div className="s-toggle">
            <span>Skip Certificate Verify</span>
            <label className="toggle">
              <input
                type="checkbox"
                checked={server.skip_verify !== false}
                onChange={(e) => updateServer({ skip_verify: e.target.checked })}
              />
              <span className="slider" />
            </label>
          </div>
        )}
      </div>

      {/* Authentication */}
      <div className="s-section">
        <div className="s-title">Authentication</div>
        <div className="s-toggle">
          <span>Use AUTH</span>
          <label className="toggle">
            <input
              type="checkbox"
              checked={server.auth}
              onChange={(e) => updateServer({ auth: e.target.checked })}
            />
            <span className="slider" />
          </label>
        </div>
        {server.auth && (
          <>
            <div className="s-field">
              <label>Auth ID</label>
              <input
                type="text"
                value={server.auth_id}
                onChange={(e) => updateServer({ auth_id: e.target.value })}
              />
            </div>
            <div className="s-field">
              <label>Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => onPasswordChange(e.target.value)}
                placeholder="OS keychain"
              />
            </div>
          </>
        )}
      </div>

      {/* Send Options */}
      <div className="s-section">
        <div className="s-title">Send Options</div>
        <div className="s-row">
          <div className="s-field">
            <label>Count</label>
            <input
              type="text"
              inputMode="numeric"
              value={countText}
              onChange={(e) => {
                setCountText(e.target.value);
                const v = parseInt(e.target.value);
                if (!isNaN(v) && v >= 1 && v <= 100000) updateMail({ mail_number: v });
              }}
              onBlur={() => {
                const v = parseInt(countText);
                if (isNaN(v) || v < 1) {
                  updateMail({ mail_number: 1 });
                  setCountText('1');
                } else if (v > 100000) {
                  updateMail({ mail_number: 100000 });
                  setCountText('100000');
                }
              }}
            />
          </div>
          <div className="s-field">
            <label>Threads</label>
            <input
              type="text"
              inputMode="numeric"
              value={threadsText}
              onChange={(e) => {
                setThreadsText(e.target.value);
                const v = parseInt(e.target.value);
                if (!isNaN(v) && v >= 1 && v <= 50) updateMail({ thread_number: v });
              }}
              onBlur={() => {
                const v = parseInt(threadsText);
                if (isNaN(v) || v < 1) {
                  updateMail({ thread_number: 1 });
                  setThreadsText('1');
                } else if (v > 50) {
                  updateMail({ thread_number: 50 });
                  setThreadsText('50');
                }
              }}
            />
          </div>
        </div>
        <div className="s-field">
          <label>Interval (ms)</label>
          <input
            type="text"
            inputMode="numeric"
            value={intervalText}
            onChange={(e) => {
              setIntervalText(e.target.value);
              const v = parseInt(e.target.value);
              if (!isNaN(v) && v >= 0 && v <= 60000) updateMail({ interval_ms: v });
              else if (e.target.value === '') updateMail({ interval_ms: 0 });
            }}
            onBlur={() => {
              const v = parseInt(intervalText);
              if (isNaN(v) || v < 0) {
                updateMail({ interval_ms: 0 });
                setIntervalText('0');
              } else if (v > 60000) {
                updateMail({ interval_ms: 60000 });
                setIntervalText('60000');
              }
            }}
          />
        </div>
      </div>

      {/* Custom Headers */}
      <div className="s-section">
        <div className="s-title">Custom Headers</div>
        <div className="header-list">
          {(mail.custom_headers || []).map((h, i) => (
            <div className="header-item" key={i}>
              <input
                type="text"
                value={h.key}
                onChange={(e) => updateHeader(i, 'key', e.target.value)}
                placeholder="Name"
                style={{ flex: '0 0 80px' }}
              />
              <input
                type="text"
                value={h.value}
                onChange={(e) => updateHeader(i, 'value', e.target.value)}
                placeholder="Value"
                style={{ flex: 1, minWidth: 0 }}
              />
              <button className="btn-remove" onClick={() => removeHeader(i)}>×</button>
            </div>
          ))}
          <button className="btn-add-header" onClick={addHeader}>+ Add Header</button>
        </div>
      </div>
    </div>
  );
}
