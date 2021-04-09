import {assert} from 'chai';
import {renderMarkdown, shouldRenderMarkdown} from './md-helper.js';

describe('shouldRenderMarkdown', () => {
  it('defaults to false', () => {
    const actual = shouldRenderMarkdown();
    assert.isFalse(actual);
  });

  it('can be overriden to true', () => {
    const actual = shouldRenderMarkdown({override: true});
    assert.isTrue(actual);
  });
});

describe('renderMarkdown', () => {
  it('can render empty string', () => {
    const actual = renderMarkdown('');
    assert.equal(actual.toString(), '');
  });

  it('can render basic string', () => {
    const actual = renderMarkdown('hello world');
    assert.equal(actual.toString(), '<p>hello world</p>\n');
  });

  it('can render lists', () => {
    const input = '* First item\n* Second item\n* Third item\n* Fourth item';
    const actual = renderMarkdown(input);
    const expected = '<ul>\n<li>First item</li>\n<li>Second item</li>\n' +
        '<li>Third item</li>\n<li>Fourth item</li>\n</ul>\n';
    assert.equal(actual.toString(), expected);
  });

  it('can render headings', () => {
    const actual = renderMarkdown('# Heading level 1\n\n## Heading level 2');
    assert.equal(actual.toString(),
        '<h1>Heading level 1</h1>\n<h2>Heading level 2</h2>\n');
  });

  it('can render links', () => {
    const actual = renderMarkdown('[clickme](http://google.com)');
    assert.equal(actual.toString(),
        '<p><a href="http://google.com">clickme</a></p>\n');
  });

  it('preserves bolding from description templates', () => {
    const input = `<b>What's the problem?</b>\n<b>1.</b> A\n<b>2.</b> B`;
    const actual = renderMarkdown(input);
    const expected = `<p><strong>What's the problem?</strong>\n<strong>1.` +
        `</strong> A\n<strong>2.</strong> B</p>\n`;
    assert.equal(actual.toString(), expected);
  });
});
