'use strict';

// Default snooze times per tree in minutes.
const DefaultSnoozeTimes = {
  'chromium.perf': 60 * 24,
  '*': 60,
};

const ONE_MIN_MS = 1000 * 60;

// TODO(yuanzhi) Consider construct this list from a monorail query.
const MONORAIL_PROJECTS = ['chromium',
  'fuchsia', 'gn', 'monorail', 'v8', 'webrtc'];

class SomAnnotations extends Polymer.mixinBehaviors([
  AnnotationManagerBehavior,
  PostBehavior,
  AlertTypeBehavior,
  BugManagerBehavior,
],
    Polymer.Element) {
  static get is() {
    return 'som-annotations';
  }

  static get properties() {
    return {
      // All alert annotations. Includes values from localState.
      annotations: {
        notify: true,
        type: Object,
        value: function() {
          return {};
        },
        computed: '_computeAnnotations(_annotationsResp)',
      },
      annotationError: {
        type: Object,
        value: function() {
          return {};
        },
      },
      // The raw response from the server of annotations.
      _annotationsResp: {
        type: Array,
        value: function() {
          return [];
        },
      },
      _bugErrorMessage: String,
      _fileBugInput: Object,
      collapseByDefault: Boolean,
      _commentInFlight: Boolean,
      _commentsErrorMessage: String,
      _commentsModel: Object,
      _commentsModelAnnotation: {
        type: Object,
        computed:
            '_computeCommentsModelAnnotation(annotations, _commentsModel)',
      },
      _commentsHidden: {
        type: Boolean,
        computed: '_computeCommentsHidden(_commentsModelAnnotation)',
      },
      _commentTextInput: Object,
      _commentIndexToRemove: Number,
      _defaultSnoozeTime: {
        type: Number,
        computed: '_computeDefaultSnoozeTime(tree.name)',
      },
      _fileBugErrorMessage: String,
      _fileBugModel: Object,
      _fileBugCallback: Function,
      _filedBug: {
        type: Boolean,
        value: false,
      },
      _groupErrorMessage: String,
      _groupName: String,
      _groupModel: Object,
      _groupCallback: Function,
      _removeBugErrorMessage: String,
      _removeBugModel: Object,
      _snoozeErrorMessage: String,
      _snoozeModel: Object,
      _snoozeCallback: Function,
      _snoozeTimeInput: Object,
      tree: {
        type: Object,
        value: function() {
          return {};
        },
      },
      _ungroupErrorMessage: String,
      _ungroupModel: Object,
      _isGrouping: Boolean,
      user: String,
    };
  }

  ready() {
    super.ready();

    this._fileBugInput = this.$.bug;
    this._commentTextInput = this.$.commentText;
    this._snoozeTimeInput = this.$.snoozeTime;

    this.fetch();
  }

  fetch() {
    this.annotationError.action = 'Fetching all annotations';
    this.fetchAnnotations().catch((error) => {
      this.annotationError.message = error;
      this.notifyPath('annotationError.message');
    });
  }

  // Fetches new annotations from the server.
  fetchAnnotations() {
    return window.fetch('/api/v1/annotations/' +
      this.tree.name, {credentials: 'include'})
        .then(jsonParsePromise)
        .then((req) => {
          this._annotationsResp = [];
          this._annotationsResp = req;
        });
  }

  // Send an annotation change. Also updates the in memory annotation
  // database.
  // Returns a promise of the POST request to the server to carry out the
  // annotation change.
  sendAnnotation(key, type, change) {
    change.key = key;
    return this
        .postJSON('/api/v1/annotations/' + this.tree.name + '/' + type, change)
        .then(jsonParsePromise)
        .then(this._postResponse.bind(this));
  }

  _computeAnnotations(annotationsJson) {
    const annotations = {};
    annotationsJson = annotationsJson || [];

    annotationsJson.forEach((ann) => {
      const key = decodeURIComponent(ann.key);
      annotations[key] = ann;
    });
    return annotations;
  }

  _haveAnnotationError(annotationError) {
    return !!annotationError.base.message;
  }

  // Handle the result of the response of a post to the server.
  _postResponse(response) {
    const annotations = this.annotations;
    annotations[decodeURIComponent(response.key)] = response;
    const newArray = [];
    Object.keys(annotations).forEach((k) => {
      k = decodeURIComponent(k);
      newArray.push(annotations[k]);
    });
    this._annotationsResp = newArray;

    return response;
  }

  // //////////////////// Handlers ///////////////////////////

  handleAnnotation(alert, detail) {
    this.annotationError.action = 'Fetching all annotations';
    this.sendAnnotation(alert.key, detail.type, detail.change)
        .then((response) => {})
        .catch((error) => {
          this.annotationError.message = error;
          this.notifyPath('annotationError.message');
        });
  }

  handleComment(alert) {
    this._commentsModel = alert;
    this._commentsErrorMessage = '';
    this.$.commentsDialog.open();
  }

  handleLinkBug(alerts, callback) {
    this._fileBugCallback = callback;
    this._fileBugModel = alerts;

    this._bugErrorMessage = '';

    const autosnoozeTime = parseInt(this.$.autosnoozeTime.value, 10);
    this.$.autosnoozeTime.value = autosnoozeTime || this._defaultSnoozeTime;
    this.$.bugDialog.open();
  }

  handleFileBug(alerts, callback) {
    this._fileBugCallback = callback;
    this._fileBugModel = alerts;

    let bugSummary = 'Bug filed from Sheriff-o-Matic';

    if (alerts) {
      if (alerts.length > 1) {
        bugSummary = `${alerts[0].title} and ${alerts.length - 1} other alerts`;
      } else if (alerts.length) {
        bugSummary = alerts[0].title;
      }
    }

    this.$.fileBug.summary = bugSummary;
    this.$.fileBug.description = this._commentForBug(this._fileBugModel);
    this.$.fileBug.labels = this._computeFileBugLabels(this.tree, alerts);
    this.$.fileBug.projectId = this.tree.default_monorail_project_name;
    this.$.fileBug.open();
  }

  handleRemoveBug(alert, detail) {
    this.$.removeBugDialog.open();
    this._removeBugModel = Object.assign({alert: alert}, detail);
    this._removeBugErrorMessage = '';
  }

  handleSnooze(alerts, callback) {
    this._snoozeCallback = callback;
    this._snoozeModel = alerts;
    this._snoozeErrorMessage = '';
    this.$.snoozeDialog.open();
  }

  handleGroupAlerts(alerts, callback) {
    this._isGrouping = false;
    this._groupModel = alerts;
    this._groupErrorMessage = '';
    this._groupCallback = callback;
    this._groupName = '';
    for (const alert of alerts) {
      if (alert.grouped) {
        this._groupName = alert.title;
      }
    }
    this.$.groupDialog.open();
  }

  handleUngroup(alert) {
    this._ungroupModel = alert;
    this._ungroupErrorMessage = '';
    this.$.ungroupDialog.open();
  }

  handleUngroupBulk(groupedAlerts) {
    this._ungroupBulkModel = groupedAlerts;
    this._ungroupBulkErrorMessage = '';
    this.$.ungroupBulkDialog.open();
  }

  // //////////////////// Bugs ///////////////////////////

  _linkNewBug() {
    // TODO(zhangtiff): Move annotation creation to the backend.
    const data = {
      bugs: [{
        id: this.$.fileBug.filedBugId,
        projectId: this.$.fileBug.projectId,
      }],
    };

    const promises = this._fileBugModel.map((alert) => {
      return this.sendAnnotation(alert.key, 'add', data);
    });
    Promise.all(promises).then(
        (response) => {
          this.$.fileBug.onBugLinkedSuccessfully();

          if (this._fileBugCallback) {
            this._fileBugCallback();
          }
        },
        (error) => {
          this.$.fileBug.onBugLinkedFailed(error);
        });
    return response;
  }

  _builderFailureInfo(builder) {
    let s = 'Builder: ' + builder.name;
    s += '\n' + builder.url;
    if (builder.first_failure_url) {
      s += '\n' +
           'First failing build:';
      s += '\n' + builder.first_failure_url;
    } else if (builder.latest_failure_url) {
      s += '\n' +
           'Latest failing build:';
      s += '\n' + builder.latest_failure_url;
    }
    return s;
  }

  _commentForBug(alerts) {
    return alerts.reduce((comment, alert) => {
      let result = alert.title + '\n\n';
      if (alert.extension) {
        if (alert.extension.builders && alert.extension.builders.length > 0) {
          const failuresInfo = [];
          for (const builder of alert.extension.builders) {
            failuresInfo.push(this._builderFailureInfo(builder));
          }
          result += 'List of failed builders:\n\n' +
                    failuresInfo.join('\n--------------------\n') + '\n\n';
        }
        if (alert.extension.reasons && alert.extension.reasons.length > 0) {
          result += 'Reasons: ';
          for (let i = 0; i < alert.extension.reasons.length; i++) {
            result += '\n' + alert.extension.reasons[i].url;
            if (alert.extension.reasons[i].test_names) {
              result += '\n' +
                        'Tests:';
              if (alert.extension.reasons[i].test_names) {
                result +=
                    '\n* ' + alert.extension.reasons[i].test_names.join('\n* ');
              }
            }
          }
          result += '\n\n';
        }
      }
      return comment + result;
    }, '');
  }

  _fileBugClicked() {
    this.$.bugDialog.close();
    this.handleFileBug(this._fileBugModel, this._fileBugCallBack);
  }

  _removeBug() {
    const model = this._removeBugModel;
    const data = {
      bugs: [{
        id: model.bug,
        projectId: model.project,
      }],
    };
    this.sendAnnotation(model.alert.key, 'remove', data)
        .then(
            (response) => {
              this.$.removeBugDialog.close();
              this._removeBugErrorMessage = '';
            },
            (error) => {
              this._removeBugErrorMessage = error;
            });
  }

  // Parse URL to get bug id if its in the query segment.
  // ex: crbug.com/issues/detail?id=123 will return 123
  _getBugIDFromURLQuery(url) {
    const _params = new URLSearchParams(url.search);
    if (_params.has('id') && !isNaN(parseInt(_params.get('id')))) {
      return _params.get('id');
    }
    return NaN;
  }

  // Parse URL to get bug id if its the last path segment
  // ex: crbug.com/123 will return 123
  _getBugIDFromURLPath(url) {
    const pathName = url.pathname;
    const bugId = pathName.substring(pathName.lastIndexOf('/') + 1);
    return isNaN(parseInt(bugId)) ? NaN : bugId;
  }

  // Parse URL to read back bug id as a numerical string.
  _getBugIDFromURL(url) {
    const bugId = this._getBugIDFromURLPath(url);
    return isNaN(bugId) ? this._getBugIDFromURLQuery(url) : bugId;
  }

  // Parse for input if its in the format
  // project:bug (ex: chromium:1234)
  _getBugIDFromString(input) {
    const fields = input.split(':');
    if (!input.startsWith('http') && fields.length == 2 &&
        MONORAIL_PROJECTS.includes(fields[0])) {
      return isNaN(parseInt(fields[1])) ? [] : fields;
    }
    return [];
  }

  // Parse the url object to read back the project name
  // ex: crbug.com/p/monorail/123 will return monorail
  // ex: crbug.com/123 will return chromium
  // ex: crbug.com/monorail/123 will return monorail
  // ex: bugs.chromium.org/p/monorail/issues/detail?id=1024028
  // will return monorail
  _getProjectNameFromUrl(url) {
    const paths = url.pathname.split('/');
    if (paths.length > 2) {
      if (url.hostname == 'bugs.chromium.org') {
        return paths[2];
      }
      return paths[paths.length - 2];
    }
    return 'chromium';
  }

  // Checks if url begins with http or https, if not, prepend "http://".
  _cleanupUrl(url) {
    if (!/^https?:\/\//i.test(url)) {
      url = 'http://' + url;
    }
    return url;
  }

  _getBugDataFromURL(url) {
    let projectName = '';
    let bugID = '';

    // Check if input is in the format "chromium:1234"
    const bugInfo = this._getBugIDFromString(url);
    if (bugInfo.length == 2) {
      projectName = bugInfo[0];
      bugID = bugInfo[1];
    } else if (!isNaN(parseInt(url))) {
      // If input is numerical, default to chromium project.
      projectName = 'chromium';
      bugID = url;
    } else {
      // If input is url, parse both path and query fields for
      // project and bug id.
      const _url = new URL(this._cleanupUrl(url));
      switch (_url.hostname) {
        case 'fxb':
        case 'bugs.fuchsia.dev':
          projectName = 'fuchsia';
          break;
        case 'crbug.com':
        case 'crbug':
        case 'bugs.chromium.org':
          // Parse url path for project
          projectName = this._getProjectNameFromUrl(_url);
          break;
        default:
          projectName = 'chromium';
      }
      bugID = this._getBugIDFromURL(_url);
      if (isNaN(bugID)) {
        throw Error('Input ' + url + ' is not a valid Bug ID or URL. ' +
          'Allowed formats are: \n' +
          ' <id> (ex:1234, defaults to chromium project) \n' +
          ' <project>:<id> (ex: chromium:1234) \n' +
          ' <hostName>/<id> (ex: bugs.chromium.org/1234) \n' +
          ' <shortName>/<id> (ex: crbugs.com/1234) \n' +
          ' <FullURL> (ex: https://bugs.fuchsia.dev/p/fuchsia/issues/detail?id=1234) \n');
      }
    }
    return {
      bugs: [{
        id: bugID,
        projectId: projectName,
      }],
    };
  }

  _saveBug() {
    let data;
    try {
      data = this._getBugDataFromURL(this.$.bug.value.trim());
    } catch (e) {
      this._bugErrorMessage = e;
      return;
    }
    if (this.$.autosnooze.checked) {
      const autosnoozeTime = parseInt(this.$.autosnoozeTime.value, 10);
      if (isNaN(autosnoozeTime)) {
        this._bugErrorMessage = 'Please enter a valid snooze time.';
        return;
      }
      const snoozeTime = autosnoozeTime || this._defaultSnoozeTime;
      data.snoozeTime = Date.now() + ONE_MIN_MS * snoozeTime;
    }
    const promises = this._fileBugModel.map((alert) => {
      return this.sendAnnotation(alert.key, 'add', data);
    });
    Promise.all(promises).then(
        (response) => {
          this._bugErrorMessage = '';
          this.$.bug.value = '';
          this.$.bugDialog.close();

          if (this._fileBugCallback) {
            this._fileBugCallback();
          }
        },
        (error) => {
          this._bugErrorMessage = error;
        });
  }

  // //////////////////// Snooze ///////////////////////////

  _snooze() {
    const promises = this._snoozeModel.map((alert) => {
      return this.sendAnnotation(alert.key, 'add', {
        snoozeTime: Date.now() + ONE_MIN_MS * this.$.snoozeTime.value,
      });
    });
    Promise.all(promises).then(
        (response) => {
          this.$.snoozeDialog.close();

          if (this._snoozeCallback) {
            this._snoozeCallback();
          }
        },
        (error) => {
          this._snoozeErrorMessage = error;
        });
  }

  // //////////////////// Comments ///////////////////////////

  _addComment() {
    if (this._commentInFlight) {
      return;
    }

    const text = this.$.commentText.value;
    if (!(text && /[^\s]/.test(text))) {
      this._commentsErrorMessage = 'Comment text cannot be blank!';
      return;
    }
    this._commentInFlight = true;
    this.sendAnnotation(this._commentsModel.key, 'add', {
      comments: [text],
    })
        .then(
            (response) => {
              this.$.commentText.value = '';
              this._commentsErrorMessage = '';
              this._commentInFlight = false;
            },
            (error) => {
              this._commentsErrorMessage = error;
              this._commentInFlight = false;
            });
  }

  _computeCommentsHidden(annotation) {
    return !(annotation && annotation.comments);
  }

  // This is mostly to make sure the comments in the modal get updated
  // properly if changed.
  _computeCommentsModelAnnotation(annotations, model) {
    if (!annotations || !model) {
      return null;
    }
    return this.computeAnnotation(annotations, model, this.collapseByDefault);
  }

  _computeDefaultSnoozeTime(treeName) {
    if (treeName in DefaultSnoozeTimes) {
      return DefaultSnoozeTimes[treeName];
    }
    return DefaultSnoozeTimes['*'];
  }

  _computeFileBugLabels(tree, alerts) {
    const labels = ['Filed-Via-SoM'];
    if (!tree) {
      return labels;
    }

    // TODO(zhangtiff): Replace this with some way to mark internal trees and
    // automatically add RVG labels to them.
    if (tree.name === 'android') {
      labels.push('Restrict-View-Google');
    }
    if (tree.bug_queue_label) {
      labels.push(tree.bug_queue_label);
    }

    if (alerts) {
      const trooperBug = alerts.some((alert) => {
        return this.isTrooperAlertType(alert.type);
      });

      if (trooperBug) {
        labels.push('Infra-Troopers');
      }
    }
    return labels;
  }

  _computeHideDeleteComment(comment) {
    return comment.user != this.user;
  }

  _computeUsername(email) {
    if (!email) {
      return email;
    }
    const cutoff = email.indexOf('@');
    if (cutoff < 0) {
      return email;
    }
    return email.substring(0, cutoff);
  }

  _formatTimestamp(timestamp) {
    if (!timestamp) {
      return '';
    }
    const time = moment.tz(new Date(timestamp), 'Atlantic/Reykjavik');
    const result =
        time.tz('America/Los_Angeles').format('ddd, DD MMM Y hh:mm A z');
    return result + ` (${time.fromNow()})`;
  }

  _confirmRemoveComment(evt) {
    this._commentIndexToRemove = evt.model.comment.index;
    this.$.removeCommentConfirmationDialog.open();
  }

  _removeComment(evt) {
    this.$.removeCommentConfirmationDialog.close();
    const request = this.sendAnnotation(this._commentsModel.key, 'remove', {
      comments: [this._commentIndexToRemove],
    });
    if (request) {
      request.then((response) => {}, (error) => {
        this._commentsErrorMessage = error;
      });
    }
  }

  // //////////////////// Groups ///////////////////////////

  _group() {
    const groupName = this.$.groupName.value.trim();
    if (!groupName) {
      this._groupErrorMessage = 'Please enter a group name';
      return;
    }
    const shouldMergeBugs = this.$.mergeBugs.checked;
    this.group(this._groupModel, groupName, shouldMergeBugs);
  }

  group(alerts, groupName, shouldMergeBugs) {
    this._groupErrorMessage = '';
    const changes = [];

    // Determine group ID.
    let groupAlert = null;
    for (const i in alerts) {
      if (alerts[i].grouped) {
        if (groupAlert) {
          this._groupErrorMessage = 'attempting to group multiple groups';
          return;
        }
        groupAlert = alerts[i];
      }
    }
    const groupID = groupAlert ? groupAlert.key : this._generateUUID();

    // Determine ungrouped alerts to group.
    alerts = alerts.filter((a) => {
      return !a.grouped;
    });

    // Data cleanup: If the group is resolved, ensure that all subalerts
    // are resolved too.
    if (groupAlert && groupAlert.resolved) {
      for (let i = 0; i < groupAlert.alerts.length; i++) {
        const subAlert = groupAlert.alerts[i];
        if (!subAlert.resolved) {
          this._groupModel.resolveAlerts([subAlert], true);
        }
      }
    } else if (groupAlert && !groupAlert.resolved) {
      for (const i in alerts) {
        if (alerts[i].resolved) {
          this._groupErrorMessage =
              'attempting to group resolved alert with unresolved group';
          return;
        }
      }
    }

    this._isGrouping = true;
    // Create annotation for each ungrouped alert key.
    for (let i = 0; i < alerts.length; i++) {
      // Grouping with a resolved group will resolve all unresolved issues.
      if (groupAlert && groupAlert.resolved && !alerts[i].resolved) {
        this._groupModel.resolveAlerts([alerts[i]], true);
      }

      if (this._groupErrorMessage) {
        break;
      }
      changes.push(
          this.sendAnnotation(alerts[i].key, 'add', {group_id: groupID}));
    }

    const groupChanges = {group_id: groupName};
    if (shouldMergeBugs) {
      groupChanges['bugs'] =
          alerts.map((alert) => this.computeAnnotation(this.annotations, alert))
              .map((annotation) => this.computeBugs(annotation))
              .flat()
              .map((bug) => ({
                id: bug.id.toString(),
                projectId: bug.projectId,
              }));
    }

    changes.push(this.sendAnnotation(groupID, 'add', groupChanges));
    Promise.all(changes).then(
        (resps) => {
          this.$.groupDialog.close();
          this._groupCallback();
          this._isGrouping = false;
        },
        (error) => {
          this._groupErrorMessage = error;
          this._isGrouping = false;
        });
  }

  _ungroup() {
    // TODO(add proper error handling)
    for (const i in this._ungroupModel.alerts) {
      if (!this._ungroupErrorMessage && this._ungroupModel.alerts[i].checked) {
        this.sendAnnotation(this._ungroupModel.alerts[i].key, 'remove',
            {group_id: true})
            .then(
                (response) => {
                  this.$.ungroupDialog.close();
                },
                (error) => {
                  this._ungroupErrorMessage = error;
                });
        // TODO(davidriley): Figure out why things remain checked.
        this._ungroupModel.alerts[i].checked = false;
      }
    }
  }

  _ungroupBulk() {
    // TODO(add proper error handling)
    const ungroupedAlerts = [];
    const changes = [];
    for (let i = 0; i < this._ungroupBulkModel.length; i++) {
      const group = this._ungroupBulkModel[i];
      if (!this._ungroupBulkErrorMessage && group.checked) {
        for (let j = 0; j < group.alerts.length; j++) {
          const alert = group.alerts[j];
          changes.push(
              this.sendAnnotation(alert.key, 'remove', {group_id: true}));
          ungroupedAlerts.push(alert);
        }
      }
      // TODO(seanmccullough): Figure out why things remain checked.
      group.checked = false;
    }

    Promise.all(changes).then(
        (resps) => {
          this.$.ungroupBulkDialog.close();
          this.fire('bulk-ungrouped', ungroupedAlerts);
        },
        (error) => {
          this._ungroupBulkErrorMessage = error;
        });
  }

  _haveSubAlerts(alert) {
    return alert.alerts && alert.alerts.length > 0;
  }

  _haveStages(alert) {
    return alert.extension && alert.extension.stages &&
           alert.extension.stages.length > 0;
  }

  _generateUUID() {
    // This is actually an rfc4122 version 4 compliant uuid taken from:
    // http://stackoverflow.com/questions/105034
      return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(
        /[xy]/g, function (c) {
          var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
          return v.toString(16);
        });
  }

  // //////////////////// Misc UX ///////////////////////////

  _checkAll(e) {
    const target = e.target;
    const checkboxSelector = target.getAttribute('data-checkbox-selector');
    const checkboxes =
        Polymer.dom(this.root).querySelectorAll(checkboxSelector);
    for (let i = 0; i < checkboxes.length; i++) {
      // Note: We are using .click() because otherwise the checkbox's change
      // event is not fired.
      if (checkboxes[i].checked != target.checked) {
        checkboxes[i].click();
      }
    }
  }
}

customElements.define(SomAnnotations.is, SomAnnotations);
