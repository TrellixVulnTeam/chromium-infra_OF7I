// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {LitElement, html, css} from 'lit-element';

import 'elements/framework/mr-upload/mr-upload.js';
import 'elements/framework/mr-error/mr-error.js';
import {fieldTypes} from 'shared/issue-fields.js';
import {store, connectStore} from 'reducers/base.js';
import * as issueV0 from 'reducers/issueV0.js';
import * as projectV0 from 'reducers/projectV0.js';
import * as userV0 from 'reducers/userV0.js';
import 'elements/chops/chops-checkbox/chops-checkbox.js';
import 'elements/chops/chops-dialog/chops-dialog.js';
import {SHARED_STYLES, MD_PREVIEW_STYLES, MD_STYLES} from 'shared/shared-styles.js';
import {commentListToDescriptionList} from 'shared/convertersV0.js';
import {renderMarkdown, shouldRenderMarkdown} from 'shared/md-helper.js';
import {unsafeHTML} from 'lit-html/directives/unsafe-html.js';


/**
 * `<mr-edit-description>`
 *
 * A dialog to edit descriptions.
 *
 */
export class MrEditDescription extends connectStore(LitElement) {
  /** @override */
  constructor() {
    super();
    this._editedDescription = '';
  }

  /** @override */
  static get styles() {
    return [
      SHARED_STYLES,
      MD_PREVIEW_STYLES,
      MD_STYLES,
      css`
        chops-dialog {
          --chops-dialog-width: 800px;
        }
        textarea {
          font-family: var(--mr-toggled-font-family);
          min-height: 300px;
          max-height: 500px;
          border: var(--chops-accessible-border);
          padding: 0.5em 4px;
          margin: 0.5em 0;
        }
        .attachments {
          margin: 0.5em 0;
        }
        .content {
          padding: 0.5em 0.5em;
          width: 100%;
          box-sizing: border-box;
        }
        .edit-controls {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }
      `,
    ];
  }

  /** @override */
  render() {
    return html`
      <link href="https://fonts.googleapis.com/icon?family=Material+Icons"
            rel="stylesheet">
      <chops-dialog aria-labelledby="editDialogTitle">
        <h3 id="editDialogTitle" class="medium-heading">
          Edit ${this._title}
        </h3>
        <textarea
          id="description"
          class="content"
          @keyup=${this._setEditedDescription}
          @beforeinput=${this._setEditedDescription}
          @change=${this._setEditedDescription}
          .value=${this._editedDescription}
        ></textarea>
        ${this._renderMarkdown ? html`
          <div class="markdown-preview preview-height-description">
            <div class="markdown">
              ${unsafeHTML(renderMarkdown(this._editedDescription))}
            </div>
          </div>`: ''}
        <h3 class="medium-heading">
          Add attachments
        </h3>
        <div class="attachments">
          ${this._attachments && this._attachments.map((attachment) => html`
            <label>
              <chops-checkbox
                type="checkbox"
                checked="true"
                class="kept-attachment"
                data-attachment-id=${attachment.attachmentId}
                @checked-change=${this._keptAttachmentIdsChanged}
              />
              <a href=${attachment.viewUrl} target="_blank">
                ${attachment.filename}
              </a>
            </label>
            <br>
          `)}
          <mr-upload></mr-upload>
        </div>
        <mr-error
          ?hidden=${!this._attachmentError}
        >${this._attachmentError}</mr-error>
        <div class="edit-controls">
          <chops-checkbox
            id="sendEmail"
            ?checked=${this._sendEmail}
            @checked-change=${this._setSendEmail}
          >Send email</chops-checkbox>
          <div>
            <chops-button id="discard" @click=${this.cancel} class="de-emphasized">
              Discard
            </chops-button>
            <chops-button id="save" @click=${this.save} class="emphasized">
              Save changes
            </chops-button>
          </div>
        </div>
      </chops-dialog>
    `;
  }

  /** @override */
  static get properties() {
    return {
      commentsByApproval: {type: Array},
      issueRef: {type: Object},
      fieldName: {type: String},
      projectName: {type: String},
      _attachmentError: {type: String},
      _attachments: {type: Array},
      _boldLines: {type: Array},
      _editedDescription: {type: String},
      _title: {type: String},
      _keptAttachmentIds: {type: Object},
      _sendEmail: {type: Boolean},
      _prefs: {type: Object},
    };
  }

  /** @override */
  stateChanged(state) {
    this.commentsByApproval = issueV0.commentsByApprovalName(state);
    this.issueRef = issueV0.viewedIssueRef(state);
    this.projectName = projectV0.viewedProjectName(state);
    this._prefs = userV0.prefs(state);
  }

  /**
   * Public function to open the issue description editing dialog.
   * @param {Event} e
   */
  async open(e) {
    await this.updateComplete;
    this.shadowRoot.querySelector('chops-dialog').open();
    this.fieldName = e.detail.fieldName;
    this.reset();
  }

  /**
   * Resets edit form.
   */
  async reset() {
    await this.updateComplete;
    this._attachmentError = '';
    this._attachments = [];
    this._boldLines = [];
    this._keptAttachmentIds = new Set();

    const uploader = this.shadowRoot.querySelector('mr-upload');
    if (uploader) {
      uploader.reset();
    }

    // Sets _editedDescription and _title.
    this._initializeView(this.commentsByApproval, this.fieldName);

    this.shadowRoot.querySelectorAll('.kept-attachment').forEach((checkbox) => {
      checkbox.checked = true;
    });
    this.shadowRoot.querySelector('#sendEmail').checked = true;

    this._sendEmail = true;
  }

  /**
   * Cancels in-flight edit data.
   */
  async cancel() {
    await this.updateComplete;
    this.shadowRoot.querySelector('chops-dialog').close();
  }

  /**
   * Sends the user's edit to Monorail's backend to be saved.
   */
  async save() {
    const commentContent = this._markupNewContent();
    const sendEmail = this._sendEmail;
    const keptAttachments = Array.from(this._keptAttachmentIds);
    const message = {
      issueRef: this.issueRef,
      isDescription: true,
      commentContent,
      keptAttachments,
      sendEmail,
    };

    try {
      const uploader = this.shadowRoot.querySelector('mr-upload');
      const uploads = await uploader.loadFiles();
      if (uploads && uploads.length) {
        message.uploads = uploads;
      }

      if (!this.fieldName) {
        store.dispatch(issueV0.update(message));
      } else {
        // This is editing an approval if there is no field name.
        message.fieldRef = {
          type: fieldTypes.APPROVAL_TYPE,
          fieldName: this.fieldName,
        };
        store.dispatch(issueV0.updateApproval(message));
      }
      this.shadowRoot.querySelector('chops-dialog').close();
    } catch (e) {
      this._attachmentError = `Error while loading file for attachment: ${
        e.message}`;
    }
  }

  /**
   * Getter for checking if the user has Markdown enabled.
   * @return {boolean} Whether Markdown preview should be rendered or not.
   */
   get _renderMarkdown() {
    const enabled = this._prefs.get('render_markdown');
    return shouldRenderMarkdown({project: this.projectName, enabled});
  }

  /**
   * Event handler for keeping <mr-edit-description>'s copy of
   * _editedDescription in sync.
   * @param {Event} e
   */
  _setEditedDescription(e) {
    const target = e.target;
    this._editedDescription = target.value;
  }

  /**
   * Event handler for keeping attachment state in sync.
   * @param {Event} e
   */
  _keptAttachmentIdsChanged(e) {
    e.target.checked = e.detail.checked;
    const attachmentId = Number.parseInt(e.target.dataset.attachmentId);
    if (e.target.checked) {
      this._keptAttachmentIds.add(attachmentId);
    } else {
      this._keptAttachmentIds.delete(attachmentId);
    }
  }

  _initializeView(commentsByApproval, fieldName) {
    this._title = fieldName ? `${fieldName} Survey` : 'Description';
    const key = fieldName || '';
    if (!commentsByApproval || !commentsByApproval.has(key)) return;
    const comments = commentListToDescriptionList(commentsByApproval.get(key));

    const comment = comments[comments.length - 1];

    if (comment.attachments) {
      this._keptAttachmentIds = new Set(comment.attachments.map(
          (attachment) => Number.parseInt(attachment.attachmentId)));
      this._attachments = comment.attachments;
    }

    this._processRawContent(comment.content);
  }

  _processRawContent(content) {
    const chunks = content.trim().split(/(<b>[^<\n]+<\/b>)/m);
    const boldLines = [];
    let cleanContent = '';
    chunks.forEach((chunk) => {
      if (chunk.startsWith('<b>') && chunk.endsWith('</b>')) {
        const cleanChunk = chunk.slice(3, -4).trim();
        cleanContent += cleanChunk;
        // Don't add whitespace to boldLines.
        if (/\S/.test(cleanChunk)) {
          boldLines.push(cleanChunk);
        }
      } else {
        cleanContent += chunk;
      }
    });

    this._boldLines = boldLines;
    this._editedDescription = cleanContent;
  }

  _markupNewContent() {
    const lines = this._editedDescription.trim().split('\n');
    const markedLines = lines.map((line) => {
      let markedLine = line;
      const matchingBoldLine = this._boldLines.find(
          (boldLine) => (line.startsWith(boldLine)));
      if (matchingBoldLine) {
        markedLine =
          `<b>${matchingBoldLine}</b>${line.slice(matchingBoldLine.length)}`;
      }
      return markedLine;
    });
    return markedLines.join('\n');
  }

  /**
   * Event handler for keeping email state in sync.
   * @param {Event} e
   */
  _setSendEmail(e) {
    this._sendEmail = e.detail.checked;
  }
}

customElements.define('mr-edit-description', MrEditDescription);
