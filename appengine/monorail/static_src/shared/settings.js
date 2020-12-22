// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// List of content type prefixes that the user will not be warned about when
// downloading an attachment.
export const ALLOWED_CONTENT_TYPE_PREFIXES = [
  'audio/', 'font/', 'image/', 'text/plain', 'video/',
];

// List of file extensions that the user will not be warned about when
// downloading an attachment.
export const ALLOWED_ATTACHMENT_EXTENSIONS = [
  '.avi', '.avif', '.bmp', '.csv', '.doc', '.docx', '.email', '.eml', '.gif',
  '.ico', '.jpeg', '.jpg', '.log', '.m4p', '.m4v', '.mkv', '.mov', '.mp2',
  '.mp4', '.mpeg', '.mpg', '.mpv', '.odt', '.ogg', '.pdf', '.png', '.sql',
  '.svg', '.tif', '.tiff', '.txt', '.wav', '.webm', '.wmv',
];

// The message to show the user when they attempt to download an unrecognized
// file type.
export const FILE_DOWNLOAD_WARNING = 'This file type is not recognized. Are' +
  ' you sure you want to download this attachment?';
