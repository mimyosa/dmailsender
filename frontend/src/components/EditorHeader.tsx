import { core } from '../../wailsjs/go/models';

interface Props {
  config: core.AppConfig;
  onChange: (cfg: core.AppConfig) => void;
  envelopeDisabled?: boolean;
  contentDisabled?: boolean;
}

export default function EditorHeader({ config, onChange, envelopeDisabled, contentDisabled }: Props) {
  const mail = config.mail;

  const update = (patch: Partial<core.MailConfig>, isEnvelope: boolean) => {
    if (isEnvelope && envelopeDisabled) return;
    if (!isEnvelope && contentDisabled) return;
    onChange(new core.AppConfig({
      ...config,
      mail: new core.MailConfig({ ...mail, ...patch }),
    }));
  };

  return (
    <div className="editor-header">
      {/* Envelope: From / To */}
      <div className="editor-section-label">Envelope</div>
      <div className={`field-row ${envelopeDisabled ? 'disabled' : ''}`}>
        <span className="field-label">From</span>
        <input
          className="field-input"
          type="text"
          value={mail.mail_from}
          onChange={(e) => update({ mail_from: e.target.value }, true)}
          placeholder="sender@example.com"
          disabled={envelopeDisabled}
        />
        <div className="field-opts">
          <span
            className={`opt-chip ${mail.numbering_mail_from ? 'on' : ''} ${envelopeDisabled ? 'chip-disabled' : ''}`}
            onClick={() => update({ numbering_mail_from: !mail.numbering_mail_from }, true)}
            title="Auto-number"
          >
            #
          </span>
        </div>
      </div>

      <div className={`field-row ${envelopeDisabled ? 'disabled' : ''}`}>
        <span className="field-label">To</span>
        <input
          className="field-input"
          type="text"
          value={mail.rcpt_to}
          onChange={(e) => update({ rcpt_to: e.target.value }, true)}
          placeholder="recipient@example.com"
          disabled={envelopeDisabled}
        />
        <div className="field-opts">
          <span
            className={`opt-chip ${mail.numbering_rcpt_to ? 'on' : ''} ${envelopeDisabled ? 'chip-disabled' : ''}`}
            onClick={() => update({ numbering_rcpt_to: !mail.numbering_rcpt_to }, true)}
            title="Auto-number"
          >
            #
          </span>
        </div>
      </div>

      {/* Content: Subject — hidden in EML mode */}
      {!contentDisabled && (
        <>
          <div className="editor-divider" />
          <div className="editor-section-label">Content</div>
          <div className="field-row">
            <span className="field-label">Subject</span>
            <input
              className="field-input"
              type="text"
              value={mail.subject}
              onChange={(e) => update({ subject: e.target.value }, false)}
              placeholder="Email subject"
            />
            <div className="field-opts">
              <span
                className={`opt-chip ${mail.numbering_subject ? 'on' : ''}`}
                onClick={() => update({ numbering_subject: !mail.numbering_subject }, false)}
                title="Auto-number"
              >
                #
              </span>
              <span
                className={`opt-chip ${mail.timestamp_subject ? 'on' : ''}`}
                onClick={() => update({ timestamp_subject: !mail.timestamp_subject }, false)}
                title="Append timestamp"
              >
                TS
              </span>
            </div>
            <select
              className="field-select"
              value={mail.content_type}
              onChange={(e) => update({ content_type: e.target.value }, false)}
            >
              <option value="text/plain">plain</option>
              <option value="text/html">html</option>
            </select>
          </div>
        </>
      )}
    </div>
  );
}
