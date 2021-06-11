import {assert} from 'chai';
import {renderMarkdown, shouldRenderMarkdown} from './md-helper.js';

describe('shouldRenderMarkdown', () => {
  it('defaults to false', () => {
    const actual = shouldRenderMarkdown();
    assert.isFalse(actual);
  });

  it('returns true for enabled projects', () => {
    const actual = shouldRenderMarkdown({project:'astor',
      enabledProjects: new Set(['astor'])});
    assert.isTrue(actual);
  });

  it('returns false for disabled projects', () => {
    const actual = shouldRenderMarkdown({project:'hazelnut',
      enabledProjects: new Set(['astor'])});
    assert.isFalse(actual);
  });

  it('user pref can disable markdown', () => {
    const actual = shouldRenderMarkdown({project:'astor',
      enabledProjects: new Set(['astor']), enabled: false});
    assert.isFalse(actual);
  });
});

describe('renderMarkdown', () => {
  it('can render empty string', () => {
    const actual = renderMarkdown('');
    assert.equal(actual, '');
  });

  it('can render basic string', () => {
    const actual = renderMarkdown('hello world');
    assert.equal(actual, '<p>hello world</p>\n');
  });

  it('can render lists', () => {
    const input = '* First item\n* Second item\n* Third item\n* Fourth item';
    const actual = renderMarkdown(input);
    const expected = '<ul>\n<li>First item</li>\n<li>Second item</li>\n' +
        '<li>Third item</li>\n<li>Fourth item</li>\n</ul>\n';
    assert.equal(actual, expected);
  });

  it('can render headings', () => {
    const actual = renderMarkdown('# Heading level 1\n\n## Heading level 2');
    assert.equal(actual,
        '<h1>Heading level 1</h1>\n<h2>Heading level 2</h2>\n');
  });

  describe('can render links', () => {
    it('for simple links', () => {
      const actual = renderMarkdown('[clickme](http://google.com)');
      const expected = `<p><span class="annotated-link"><a title="" ` +
          `href="http://google.com"><span class="material-icons link">` +
          `link</span>clickme</a><span class="tooltip">Link destination: ` +
          `http://google.com</span></span></p>\n`;
      assert.equal(actual, expected);
    });

    it('and indicates malformed link', () => {
      const actual = renderMarkdown('[clickme](google.com)');
      const expected = `<p><span class="annotated-link"><a title="" ` +
          `href="google.com"><span class="material-icons link_off">link_off` +
          `</span>clickme</a><span class="tooltip">Link may be malformed: ` +
          `google.com</span></span></p>\n`;
      assert.equal(actual, expected);
    });
  });

  it('preserves bolding from description templates', () => {
    const input = `<b>What's the problem?</b>\n<b>1.</b> A\n<b>2.</b> B`;
    const actual = renderMarkdown(input);
    const expected = `<p><strong>What's the problem?</strong>\n<strong>1.` +
        `</strong> A\n<strong>2.</strong> B</p>\n`;
    assert.equal(actual, expected);
  });

  it('escapes HTML content', () => {
    let actual = renderMarkdown('<input></input>');
    assert.equal(actual, '<p>&lt;input&gt;&lt;/input&gt;</p>\n');

    actual = renderMarkdown('<a href="https://google.com">clickme</a>');
    assert.equal(actual,
        '<p>&lt;a href="https://google.com"&gt;clickme&lt;/a&gt;</p>\n');
  });
});
