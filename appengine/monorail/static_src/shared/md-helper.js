import marked from 'marked';
import DOMPurify from 'dompurify';

/** @type {Set} Projects that defaults content as Markdown content. */
export const DEFAULT_MD_PROJECTS = new Set(['monkeyrail']);

/** @type {Set} Authors whose comments will not be rendered as Markdown. */
const BLOCKLIST = new Set(['sheriffbot@sheriffbot-1182.iam.gserviceaccount.com',
                          'sheriff-o-matic@appspot.gserviceaccount.com',
                          'sheriff-o-matic-staging@appspot.gserviceaccount.com',
                          'bugdroid1@chromium.org',
                          'bugdroid@chops-service-accounts.iam.gserviceaccount.com',
                          'gitwatcher-staging.google.com@appspot.gserviceaccount.com',
                          'gitwatcher.google.com@appspot.gserviceaccount.com']);

/**
 * Determines whether content should be rendered as Markdown.
 * @param {string} options.project Project this content belongs to.
 * @param {number} options.author User who authored this content.
 * @param {boolean} options.enabled Per-user setting for enabling Markdown.
 * @return {boolean} Whether this content should be rendered as Markdown.
 */
export const shouldRenderMarkdown = ({
  project, author, enabled = true, enabledProjects = DEFAULT_MD_PROJECTS
} = {}) => {
  if (author in BLOCKLIST) {
    return false;
  } else if (!enabled) {
    return false;
  } else if (enabledProjects.has(project)) {
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
 * Escapes HTML characters, used to render HTML blocks in Markdown. This
 * alleviates security flaws but is not the primary security barrier, that is
 * handled by DOMPurify.
 * @param {string} text Content that looks to Marked parser to contain HTML.
 * @return {string} Same text content after escaping HTML characters.
 */
const escapeHtml = (text) => {
  return text.replace(/[&<>"'`=\/]/g, (s) => {
    return HTML_ESCAPE_MAP[s];
  });
};

/**
* Checks to see if input string is a valid HTTP link.
 * @param {string} string
 * @return {boolean} Whether input string is a valid HTTP(s) link.
 */
const isValidHttpUrl = (string) => {
  let url;

  try {
    url = new URL(string);
  } catch (_exception) {
    return false;
  }

  return url.protocol === 'http:' || url.protocol === 'https:';
};

/**
 * Renderer option for Marked.
 * See https://marked.js.org/using_pro#renderer on how to use renderer.
 * @type {Object}
 */
const renderer = {
  html(text) {
    // Do not render HTML, instead escape HTML and render as plaintext.
    return escapeHtml(text);
  },
  link(href, title, text) {
    // Overrides default link rendering by adding icon and destination on hover.
    // TODO(crbug.com/monorail/9316): Add shared-styles/MD_STYLES to all
    // components that consume the markdown renderer.
    let linkIcon;
    let tooltipText;
    if (isValidHttpUrl(href)) {
      linkIcon = `<span class="material-icons link">link</span>`;
      tooltipText = `Link destination: ${href}`;
    } else {
      linkIcon = `<span class="material-icons link_off">link_off</span>`;
      tooltipText = `Link may be malformed: ${href}`;
    }
    const tooltip = `<span class="tooltip">${tooltipText}</span>`;
    return `<span class="annotated-link"><a href=${href} ` +
        `title=${title ? title : ''}>${linkIcon}${text}</a>${tooltip}</span>`;
  },
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
