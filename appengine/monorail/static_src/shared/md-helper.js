import marked from 'marked';
import DOMPurify from 'dompurify';

/** @type {Set} Projects that defaults content as Markdown content. */
const DEFAULT_MD_PROJECTS = new Set();

/** @type {Set} Authors whose comments will not be rendered as Markdown. */
const BLOCKLIST = new Set();

/**
 * Determines whether content should be rendered as Markdown.
 * @param {string} options.project Project this content belongs to.
 * @param {number} options.author User who authored this content.
 * @param {boolean} options.override Per-issue override to force Markdown.
 * @return {boolean} Whether this content should be rendered as Markdown.
 */
export const shouldRenderMarkdown = ({
  project, author, override = false,
} = {}) => {
  if (author in BLOCKLIST) {
    return false;
  } else if (override) {
    return true;
  } else if (project in DEFAULT_MD_PROJECTS) {
    return true;
  }
  return false;
};

/** @const {Object} Options for DOMPurify sanitizer */
const SANITIZE_OPTIONS = Object.freeze({
  RETURN_TRUSTED_TYPE: true,
  FORBID_TAGS: ['style'],
  FORBID_ATTR: ['style', 'autoplay'],
});

/**
 * Replaces bold HTML tags in comment with Markdown equivalent.
 * @param {string} raw Comment string as stored in database.
 * @return {string} Comment string after b tags are placed by Markdown bolding.
 */
const replaceBoldTag = (raw) => {
  return raw.replace(/<b>|<\/b>/g, '**');
};

/** @const {Object} Basic HTML character escape mapping */
const HTML_ESCAPE_MAP = Object.freeze({
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  '\'': '&#39;',
  '/': '&#x2F;',
  '`': '&#x60;',
  '=': '&#x3D;',
});

/**
 * Escapes HTML characters, used to render HTML blocks in Markdown.
 * @param {string} text Content that looks to Marked parser to contain HTML.
 * @return {string} Same text content after escaping HTML characters.
 */
const escapeHtml = (text) => {
  return text.replace(/[&<>"'`=\/]/g, (s) => {
    return HTML_ESCAPE_MAP[s];
  });
};

/** @type {Object} Renderer option for Marked */
const renderer = {
  html(text) {
    // Do not render HTML, instead escape HTML and render as plaintext.
    return escapeHtml(text);
  },
  // TODO(crbug.com/monorail/9316): Add link renderer,
};

marked.use({renderer, headerIds: false});

/**
 * Renders Markdown content into HTML.
 * @param {string} raw Content to be intepretted as Markdown.
 * @return {string} Rendered content in HTML format.
 */
export const renderMarkdown = (raw) => {
  // TODO(crbug.com/monorail/9310): Add commentReferences, projectName,
  // and revisionUrlFormat to use in conjunction with Marked's lexer for
  // autolinking.
  // TODO(crbug.com/monorail/9310): Integrate autolink
  const preprocessed = replaceBoldTag(raw);
  const converted = marked(preprocessed);
  const sanitized = DOMPurify.sanitize(converted, SANITIZE_OPTIONS);
  return sanitized.toString();
};
