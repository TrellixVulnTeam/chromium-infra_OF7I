import {LitElement, html} from '@polymer/lit-element';
import {DateTime} from 'luxon';
import * as constants from './constants';

class RotaShiftHistory extends LitElement {
  static get properties() {
    return {
      shifts: {type: constants.Shifts},
      hide: {},
    };
  }

  constructor() {
    super();
    this.hide = true;
  }

  onCallers(shift) {
    if (!shift.OnCall) {
      return;
    }
    return shift.OnCall.map((oncall, oncallIdx) => html`
             <td>${oncall.Email}</td>`);
  }

  shiftsTemplate(ss) {
    return ss.Shifts.map((i, shiftIdx) => html`
    <tr>
      <td>
      <table>
      <tbody>
        ${this.onCallers(i)}
      </tbody>
      </table>
      </td>
      <td>${DateTime.fromISO(i.StartTime, {zone: constants.zone}).toFormat(constants.timeFormat)}</td>
      <td>${DateTime.fromISO(i.EndTime, {zone: constants.zone}).toFormat(constants.timeFormat)}</td>
      <td>${i.Comment}</td>
    </tr>`);
  }

  historyShifts() {
    if (this.hide) {
      return html`
        <button type="button" @click=${() =>(this.hide = false)}>Show History
        </button>`;
    }
    if (!this.shifts || !this.shifts.SplitShifts) {
      return;
    }

    return html`<legend>Previous shifts</legend>
      ${this.shifts.SplitShifts.map((s, splitIdx) => html`
        <h4>${s.Name}</h4>
        <table id="shifts">
        <thead>
          <th>Oncallers</th>
          <th>Start</th>
          <th>End</th>
          <th>Comment</th>
        <thead>
        <tbody>
            ${this.shiftsTemplate(s)}
        </tbody>
      `)}
      <br>
      <button type="button" @click=${() => (this.hide = true)}>Hide History
      </button>
      `;
  }

  render() {
    return html`
      <style>
        #shifts {
          font-size: small;
          border-collapse: collapse;
        }

        #shifts td, #shifts th {
          border: 1px, solid #ddd;
          padding: 8px;
        }

        #shifts th {
          text-align: left;
        }

        #shifts tr:nth-child(even){background-color: hsl(0, 0%, 95%);};
      </style>
      <fieldset>
        ${this.historyShifts()}
      </fieldset>
      </table>
      `;
  }
}

customElements.define('rota-shift-history', RotaShiftHistory);
