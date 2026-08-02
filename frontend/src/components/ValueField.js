import React from 'react';
import { formatJSONText, isJSONText } from '../api';

/**
 * The value editor: a monospace textarea that keeps whatever the author typed,
 * newlines and indentation included, plus a one-click reformat that appears
 * only while the content parses as JSON.
 *
 * A textarea rather than an input because the value is stored as text and JSON
 * is the common case — a single-line box would silently discourage the
 * formatting the store is perfectly happy to keep.
 */

/** Beyond this the box stops growing and starts scrolling, so the Save button
 * never gets pushed off a phone screen by a large value. */
const MAX_ROWS = 20;

function ValueField({ id, label = 'Value', value, onChange, minRows = 4, placeholder }) {
  const canFormat = isJSONText(value);
  // Grow with the content: formatting a value is pointless if the result is
  // then shown through a four-line slot.
  const lines = (value || '').split('\n').length;
  const rows = Math.min(Math.max(lines, minRows), MAX_ROWS);

  return (
    <div className="form-group">
      <div className="form-group__head">
        <label htmlFor={id}>{label}</label>
        {canFormat && (
          <button
            type="button"
            className="btn btn--small btn--ghost"
            onClick={() => onChange(formatJSONText(value))}
          >
            Format JSON
          </button>
        )}
      </div>
      <textarea
        id={id}
        className="value-input"
        value={value}
        onChange={e => onChange(e.target.value)}
        rows={rows}
        placeholder={placeholder}
        spellCheck={false}
      />
    </div>
  );
}

export default ValueField;
