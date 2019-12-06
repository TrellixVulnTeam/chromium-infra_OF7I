import {assert} from 'chai';
import sinon from 'sinon';

import MrChart, {
  subscribedQuery,
} from 'elements/issue-list/mr-chart/mr-chart.js';
import {prpcClient} from 'prpc-client-instance.js';

let element;
let dataLoadedPromise;

const beforeEachElement = () => {
  if (element && document.body.contains(element)) {
    // Avoid setting up multiple versions of the same element.
    document.body.removeChild(element);
    element = null;
  }
  const el = document.createElement('mr-chart');
  el.setAttribute('projectName', 'rutabaga');
  dataLoadedPromise = new Promise((resolve) => {
    el.addEventListener('allDataLoaded', resolve);
  });

  document.body.appendChild(el);
  return el;
};

describe('mr-chart', () => {
  beforeEach(() => {
    window.CS_env = {
      token: 'rutabaga-token',
      tokenExpiresSec: 0,
      app_version: 'rutabaga-version',
    };
    sinon.stub(prpcClient, 'call').callsFake(async () => {
      return {
        snapshotCount: [{count: 8}],
        unsupportedField: [],
        searchLimitReached: false,
      };
    });

    element = beforeEachElement();
  });

  afterEach(async () => {
    // _fetchData is always called when the element is connected, so we have to
    // wait until all data has been loaded.
    // Otherwise prpcClient.call will be restored and we will make actual XHR
    // calls.
    await dataLoadedPromise;

    document.body.removeChild(element);

    prpcClient.call.restore();
  });

  describe('initializes', () => {
    it('renders', () => {
      assert.instanceOf(element, MrChart);
    });

    it('sets this.projectname', () => {
      assert.equal(element.projectName, 'rutabaga');
    });
  });

  describe('data loading', () => {
    beforeEach(() => {
      // Stub MrChart.makeTimestamps to return 6, not 30 data points.
      const originalMakeTimestamps = MrChart.makeTimestamps;
      sinon.stub(MrChart, 'makeTimestamps').callsFake((endDate) => {
        return originalMakeTimestamps(endDate, 1, 6);
      });
      sinon.stub(MrChart, 'getEndDate').callsFake(() => {
        return new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
      });

      // Re-instantiate element after stubs.
      element = beforeEachElement();
    });

    afterEach(() => {
      MrChart.makeTimestamps.restore();
      MrChart.getEndDate.restore();
    });

    it('makes a series of XHR calls', async () => {
      await dataLoadedPromise;
      for (let i = 0; i < 6; i++) {
        assert.deepEqual(element.values[i], {});
      }
    });

    it('sets indices and correctly re-orders values', async () => {
      await dataLoadedPromise;

      const timestampMap = new Map([
        [1540857599, 0], [1540943999, 1], [1541030399, 2], [1541116799, 3],
        [1541203199, 4], [1541289599, 5],
      ]);
      sinon.stub(MrChart.prototype, '_fetchDataAtTimestamp').callsFake(
          async (ts) => ({issues: {'Issue Count': timestampMap.get(ts)}}));

      element.endDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
      await element._fetchData();

      assert.deepEqual(element.indices, [
        '10/29/2018', '10/30/2018', '10/31/2018',
        '11/1/2018', '11/2/2018', '11/3/2018',
      ]);
      for (let i = 0; i < 6; i++) {
        assert.deepEqual(element.values[i], {'Issue Count': i});
      }
      MrChart.prototype._fetchDataAtTimestamp.restore();
    });

    it('if issue count is null, defaults to 0', async () => {
      prpcClient.call.restore();
      sinon.stub(prpcClient, 'call').callsFake(async () => {
        return {snapshotCount: [{}]};
      });
      MrChart.makeTimestamps.restore();
      sinon.stub(MrChart, 'makeTimestamps').callsFake((endDate) => {
        return [1234567, 2345678, 3456789];
      });

      await element._fetchData(new Date());
      assert.deepEqual(element.values[0], {});
    });

    it('Retrieve data under groupby feature', async () => {
      const data = new Map([['Type-1', 0], ['Type-2', 1]]);
      sinon.stub(MrChart.prototype, '_fetchDataAtTimestamp').callsFake(
          () => ({issues: data}));

      element = beforeEachElement();

      await element._fetchData(new Date());
      for (let i = 0; i < 3; i++) {
        assert.deepEqual(element.values[i], data);
      }
      MrChart.prototype._fetchDataAtTimestamp.restore();
    });
  });

  describe('start date change detection', () => {
    it('illegal query: start-date is greater than end-date', async () => {
      await element.updateComplete;

      element.startDate = new Date('2199-11-06');
      element._fetchData();

      assert.equal(element.dateRange, 90);
      assert.equal(element.frequency, 7);
      assert.equal(element.dateRangeNotLegal, true);
    });

    it('illegal query: end_date - start_date requires more than 90 queries',
        async () => {
          await element.updateComplete;

          element.startDate = new Date('2016-10-03');
          element._fetchData();

          assert.equal(element.dateRange, 90 * 7);
          assert.equal(element.frequency, 7);
          assert.equal(element.maxQuerySizeReached, true);
        });
  });

  describe('date change behavior', () => {
    it('pushes to history API via pageJS', async () => {
      sinon.stub(element, '_page');
      sinon.spy(element, '_setDateRange');
      sinon.spy(element, '_onDateChanged');
      sinon.spy(element, '_changeUrlParams');

      await element.updateComplete;

      const thirtyButton = element.shadowRoot
          .querySelector('#two-toggle').children[2];
      thirtyButton.click();

      sinon.assert.calledOnce(element._setDateRange);
      sinon.assert.calledOnce(element._onDateChanged);
      sinon.assert.calledOnce(element._changeUrlParams);
      sinon.assert.calledOnce(element._page);

      element._page.restore();
      element._setDateRange.restore();
      element._onDateChanged.restore();
      element._changeUrlParams.restore();
    });
  });

  describe('progress bar', () => {
    it('visible based on loading progress', async () => {
      // Check for visible progress bar and hidden input after initial render
      await element.updateComplete;
      const progressBar = element.shadowRoot.querySelector('progress');
      const endDateInput = element.shadowRoot.querySelector('#end-date');
      assert.isFalse(progressBar.hasAttribute('hidden'));
      assert.isTrue(endDateInput.disabled);

      // Check for hidden progress bar and enabled input after fetch and render
      await dataLoadedPromise;
      await element.updateComplete;
      assert.isTrue(progressBar.hasAttribute('hidden'));
      assert.isFalse(endDateInput.disabled);

      // Trigger another data fetch and render, but prior to fetch complete
      // Check progress bar is visible again
      element.queryParams['start-date'] = '2012-01-01';
      await element.requestUpdate('queryParams');
      await element.updateComplete;
      assert.isFalse(progressBar.hasAttribute('hidden'));

      await dataLoadedPromise;
      await element.updateComplete;
      assert.isTrue(progressBar.hasAttribute('hidden'));
    });
  });

  describe('static methods', () => {
    describe('sortInBisectOrder', () => {
      it('orders first, last, median recursively', () => {
        assert.deepEqual(MrChart.sortInBisectOrder([]), []);
        assert.deepEqual(MrChart.sortInBisectOrder([9]), [9]);
        assert.deepEqual(MrChart.sortInBisectOrder([8, 9]), [8, 9]);
        assert.deepEqual(MrChart.sortInBisectOrder([7, 8, 9]), [7, 9, 8]);
        assert.deepEqual(
            MrChart.sortInBisectOrder([1, 2, 3, 4, 5]), [1, 5, 3, 2, 4]);
      });
    });

    describe('makeTimestamps', () => {
      it('throws an error if endDate not passed', () => {
        assert.throws(() => {
          MrChart.makeTimestamps();
        }, 'endDate required');
      });
      it('returns an array of in seconds', () => {
        const endDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
        const secondsInDay = 24 * 60 * 60;

        assert.deepEqual(MrChart.makeTimestamps(endDate, 1, 6), [
          1541289599 - (secondsInDay * 5), 1541289599 - (secondsInDay * 4),
          1541289599 - (secondsInDay * 3), 1541289599 - (secondsInDay * 2),
          1541289599 - (secondsInDay * 1), 1541289599 - (secondsInDay * 0),
        ]);
      });
      it('tests frequency greater than 1', () => {
        const endDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
        const secondsInDay = 24 * 60 * 60;

        assert.deepEqual(MrChart.makeTimestamps(endDate, 2, 6), [
          1541289599 - (secondsInDay * 4),
          1541289599 - (secondsInDay * 2),
          1541289599 - (secondsInDay * 0),
        ]);
      });
      it('tests frequency greater than 1', () => {
        const endDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
        const secondsInDay = 24 * 60 * 60;

        assert.deepEqual(MrChart.makeTimestamps(endDate, 2, 7), [
          1541289599 - (secondsInDay * 6),
          1541289599 - (secondsInDay * 4),
          1541289599 - (secondsInDay * 2),
          1541289599 - (secondsInDay * 0),
        ]);
      });
    });

    describe('dateStringToDate', () => {
      it('returns null if no input', () => {
        assert.isNull(MrChart.dateStringToDate());
      });

      it('returns a new Date at EOD UTC', () => {
        const actualDate = MrChart.dateStringToDate('2018-11-03');
        const expectedDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
        assert.equal(expectedDate.getTime(), 1541289599000, 'Sanity check.');

        assert.equal(actualDate.getTime(), expectedDate.getTime());
      });
    });

    describe('getEndDate', () => {
      let clock;

      beforeEach(() => {
        clock = sinon.useFakeTimers(10000);
      });

      afterEach(() => {
        clock.restore();
      });

      it('returns parsed input date', () => {
        const input = '2018-11-03';

        const expectedDate = new Date(Date.UTC(2018, 10, 3, 23, 59, 59));
        // Time sanity check.
        assert.equal(Math.round(expectedDate.getTime() / 1e3), 1541289599);

        const actual = MrChart.getEndDate(input);
        assert.equal(actual.getTime(), expectedDate.getTime());
      });

      it('returns EOD of current date by default', () => {
        const expectedDate = new Date();
        expectedDate.setHours(23);
        expectedDate.setMinutes(59);
        expectedDate.setSeconds(59);

        assert.equal(MrChart.getEndDate().getTime(),
            expectedDate.getTime());
      });
    });

    describe('getStartDate', () => {
      let clock;

      beforeEach(() => {
        clock = sinon.useFakeTimers(10000);
      });

      afterEach(() => {
        clock.restore();
      });

      it('returns parsed input date', () => {
        const input = '2018-07-03';

        const expectedDate = new Date(Date.UTC(2018, 6, 3, 23, 59, 59));
        // Time sanity check.
        assert.equal(Math.round(expectedDate.getTime() / 1e3), 1530662399);

        const actual = MrChart.getStartDate(input);
        assert.equal(actual.getTime(), expectedDate.getTime());
      });

      it('returns EOD of current date by default', () => {
        const today = new Date();
        today.setHours(23);
        today.setMinutes(59);
        today.setSeconds(59);

        const secondsInDay = 24 * 60 * 60;
        const expectedDate = new Date(today.getTime() -
            1000 * 90 * secondsInDay);
        assert.equal(MrChart.getStartDate(undefined, today, 90).getTime(),
            expectedDate.getTime());
      });
    });

    describe('makeIndices', () => {
      it('returns dates in mm/dd/yyy format', () => {
        const timestamps = [
          1540857599, 1540943999, 1541030399,
          1541116799, 1541203199, 1541289599,
        ];
        assert.deepEqual(MrChart.makeIndices(timestamps), [
          '10/29/2018', '10/30/2018', '10/31/2018',
          '11/1/2018', '11/2/2018', '11/3/2018',
        ]);
      });
    });

    describe('getPredictedData', () => {
      it('get predicted data shown in daily', () => {
        const values = [0, 1, 2, 3, 4, 5, 6];
        const result = MrChart.getPredictedData(
            values, values.length, 3, 1, new Date('10-02-2017'));
        assert.deepEqual(result[0], ['10/4/2017', '10/5/2017', '10/6/2017']);
        assert.deepEqual(result[1], [7, 8, 9]);
        assert.deepEqual(result[2], [0, 1, 2, 3, 4, 5, 6]);
      });

      it('get predicted data shown in weekly', () => {
        const values = [0, 7, 14, 21, 28, 35, 42, 49, 56, 63, 70, 77, 84];
        const result = MrChart.getPredictedData(
            values, 91, 13, 7, new Date('10-02-2017'));
        assert.deepEqual(result[1], values.map((x) => x+91));
        assert.deepEqual(result[2], values);
      });
    });

    describe('getErrorData', () => {
      it('get error data with perfect regression', () => {
        const values = [0, 1, 2, 3, 4, 5, 6];
        const result = MrChart.getErrorData(values, values, [7, 8, 9]);
        assert.deepEqual(result[0], [7, 8, 9]);
        assert.deepEqual(result[1], [7, 8, 9]);
      });

      it('get error data with nonperfect regression', () => {
        const values = [0, 1, 3, 4, 6, 6, 7];
        const result = MrChart.getPredictedData(
            values, values.length, 3, 1, new Date('10-02-2017'));
        const error = MrChart.getErrorData(result[2], values, result[1]);
        assert.isTrue(error[0][0] > result[1][0]);
        assert.isTrue(error[1][0] < result[1][0]);
      });
    });

    describe('getSortedLines', () => {
      it('return all lines for less than n lines', () => {
        const arrayValues = [
          {label: 'line1', data: [0, 0, 1]},
          {label: 'line2', data: [0, 1, 2]},
          {label: 'line3', data: [0, 1, 0]},
          {label: 'line4', data: [4, 0, 3]},
        ];
        const expectedValues = [
          {label: 'line1', data: [0, 0, 1]},
          {label: 'line2', data: [0, 1, 2]},
          {label: 'line3', data: [0, 1, 0]},
          {label: 'line4', data: [4, 0, 3]},
        ];
        const actualValues = MrChart.getSortedLines(arrayValues, 4);
        for (let i = 0; i < 4; i++) {
          assert.deepEqual(expectedValues[i], actualValues[i]);
        }
      });

      it('return top n lines in sorted order for more than n lines',
          () => {
            const arrayValues = [
              {label: 'line1', data: [0, 0, 1]},
              {label: 'line2', data: [0, 1, 2]},
              {label: 'line3', data: [0, 4, 0]},
              {label: 'line4', data: [4, 0, 3]},
              {label: 'line5', data: [0, 2, 3]},
            ];
            const expectedValues = [
              {label: 'line5', data: [0, 2, 3]},
              {label: 'line4', data: [4, 0, 3]},
              {label: 'line2', data: [0, 1, 2]},
            ];
            const actualValues = MrChart.getSortedLines(arrayValues, 3);
            for (let i = 0; i < 3; i++) {
              assert.deepEqual(expectedValues[i], actualValues[i]);
            }
          });
    });

    describe('getGroupByFromQuery', () => {
      it('get group by label object from URL', () => {
        const input = {'groupby': 'label', 'labelprefix': 'Type'};

        const expectedGroupBy = {
          value: 'label',
          labelPrefix: 'Type',
          display: 'Type',
        };
        assert.deepEqual(MrChart.getGroupByFromQuery(input), expectedGroupBy);
      });

      it('get group by is open object from URL', () => {
        const input = {'groupby': 'open'};

        const expectedGroupBy = {value: 'open', display: 'Is open'};
        assert.deepEqual(MrChart.getGroupByFromQuery(input), expectedGroupBy);
      });

      it('get group by none object from URL', () => {
        const input = {'groupby': ''};

        const expectedGroupBy = {value: '', display: 'None'};
        assert.deepEqual(MrChart.getGroupByFromQuery(input), expectedGroupBy);
      });
    });
  });

  describe('subscribedQuery', () => {
    it('includes start and end date', () => {
      assert.isTrue(subscribedQuery.has('start-date'));
      assert.isTrue(subscribedQuery.has('start-date'));
    });

    it('includes groupby and labelprefix', () => {
      assert.isTrue(subscribedQuery.has('groupby'));
      assert.isTrue(subscribedQuery.has('labelprefix'));
    });

    it('includes q and can', () => {
      assert.isTrue(subscribedQuery.has('q'));
      assert.isTrue(subscribedQuery.has('can'));
    });
  });
});
