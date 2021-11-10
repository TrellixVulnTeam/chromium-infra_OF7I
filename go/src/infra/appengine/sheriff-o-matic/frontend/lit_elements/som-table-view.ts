import {css, html, LitElement} from 'lit';
import {ifDefined} from 'lit/directives/if-defined.js';
import {customElement, property, state} from 'lit/decorators.js';

// som-table-view fetches and displays alerts for a tree in a
// table.  This brings much higher information density than
// the traditional alert view, allowing patterns to be
// spotted more easily.
@customElement('som-table-view')
export class SomTableView extends LitElement {  
  @property() tree?: string | Tree;

  @state() alerts?: Alert[];

  connectedCallback() {
    super.connectedCallback();
    this.fetch();
  }

  treeName(): string | undefined {
    if (typeof this.tree == 'string') {
      return this.tree;
    }
    return this.tree?.name;
  }

  fetch() {
    fetch(`/api/v1/unresolved/${this.treeName()}`)
        .then(r => r.json())
        .then((alerts: Alerts) => { this.alerts = alerts.alerts; });
  }

  render() {
    if (!this.alerts) {
      return html`loading alerts...`;
    }
    return html`
      <table>
        <thead>
          <tr>
            <th>Builder</th>
            <th>Step</th>
            <th>Builds</th>
          </tr>
        </thead>
        <tbody>
          ${this.alerts.map(a => {
            const b = a.extension.builders?.[0];
            const r = a.extension.reason;
            return html`
              <tr>
                <td>${b?.name}</td>
                <td>${r?.step}</td>
                <td>${
                  b?.first_failure_build_number &&
                        b.first_failure_build_number !== b.latest_failure_build_number
                    ? html`<a href="${b.first_failure_url}">${
                        b.first_failure_build_number}</a> - `
                    : ''}
                  <a href="${ifDefined(b?.latest_failure_url)}">${
                b?.latest_failure_build_number}</a></td>
              </tr>`
          })}
        </tbody>
      </table>
      `;
  }

  static styles = css`
    :host {
      padding: 1em 16px;
    }
    table {
      width:100%;
      border-collapse: collapse;
    }
    thead {
      background-color: #000;
      color: #fff;
    }
    th {
      text-align: left;
      font-weight: normal;
    }
    th, td {
      padding: 4px 8px;
    }
    tbody tr:hover {
      background-color: #ccc;
    }
  `;
}

// Tree object provided by polymer parent page.
interface Tree {
  name: string;
  display_name: string;
}

// Alerts response objects returned from SOM server.
// Only the fields this view uses are defined here.
interface Alerts {
  alerts: Alert[];
}

interface Alert {
  extension: Extension;
}

interface Extension {
  builders?: BuilderExtension[];
  reason?: ReasonExtension;
}

interface BuilderExtension {
  name: string;
  first_failure_build_number: number; // 0 means not present
  first_failure_url: string;
  latest_failure_build_number: number;
  latest_failure_url: string;
}

interface ReasonExtension {
  step: string;
}