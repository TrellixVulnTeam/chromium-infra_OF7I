// Copyright 2013 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
 * Code for the main user-visible status page.
 */

window.onload = function() {
  document.add_new_message.message.focus();
  help_init();
  change_init();
};

/*
 * Functions for managing the help text.
 */

function help_init() {
  // Set up the help text logic.
  var message = document.add_new_message.message;
  message.onmouseover = help_show;
  message.onmousemove = help_show;
  message.onmouseout = help_hide;
  message.onkeypress = auto_submit;

  var help = document.getElementById('help');
  help.onmouseover = help_show;
  help.onmouseout = help_hide;
}

function help_show() {
  var message = document.add_new_message.message;
  var help = document.getElementById('help');
  help.style.left = message.offsetLeft + 'px';
  help.style.top = message.offsetTop + message.offsetHeight + 'px';
  help.hidden = false;
}

function help_hide() {
  var help = document.getElementById('help');
  help.hidden = true;
}

/*
 * Misc functions.
 */

// Used by the status field.
function auto_submit(e) {
  if (!e.shiftKey && e.keyCode == 13) {
    // Catch the enter key in the textarea.  Allow shift+enter to work
    // so people editing a lot of text can play around with things.
    var form = document.getElementsByName('add_new_message')[0];
    form.submit();
    return false;
  }
  return true;
}

function change_init() {
  var form = document.add_new_message;
  var message = form.message;
  var initial_message_value = message.value;

  var disable_change_button_if_same_message = function() {
    form.change.disabled = message.value == initial_message_value;
  };

  message.oninput = disable_change_button_if_same_message;
  disable_change_button_if_same_message();
}
