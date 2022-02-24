// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {makeStyles} from '@material-ui/styles';
import React from 'react';
import Button from '@material-ui/core/Button';

const userStyles = makeStyles({
  materialIcons: {
    fontFamily: 'Material Icons',
    fontWeight: 'normal',
    fontStyle: 'normal',
    fontSize: '20px',
    lineHeight: 1,
    letterSpacing: 'normal',
    textTransform: 'none',
    display: 'inline-block',
    whiteSpace: 'nowrap',
    wordWrap: 'normal',
    direction: 'ltr',
    WebkitFontFeatureSettings: 'liga',
    WebkitFontSmoothing: 'antialiased',
  },

  controls: {
    display: 'flex',
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'flex-start',
    width: '100%',
  },

  button: {
    marginRight: '8px',
    padding: '0.1em 4px',
    display: 'inline-flex',
    width: 'auto',
    cursor: 'pointer',
    border: 'var(--chops-normal-border)',
    marginLeft: 0,
  },

  inputUpload: {
    /* We need the file uploader to be hidden but still accessible. */
    opacity: 0,
    width: 0,
    height: 0,
    position: 'absolute',
    top: -9999,
    left: -9999,
  }
});

type Props = {
  files: Array<File>,
  setFiles: Function,
}

export default function AttachmentUploader(props: Props): React.ReactElement {
  const classes = userStyles();
  const {files, setFiles} = props;

  const onSelectFile = (event: {currentTarget: any;}) => {
    const input = event.currentTarget;
    if (!input.files || input.files.length === 0) {
      return;
    }
    const newFiles = [...input.files].filter((f1) => {
      //TODO: check duplicate file
      return true;
    })

    const expendFiles = [...files].concat(newFiles);
    setFiles(expendFiles);
  }

  const onRemoveFile = (index: number) => () => {
    let remainingFiles = [...files];
    remainingFiles.splice(index, 1);
    setFiles(remainingFiles);
  }
  return (
    <>
      <div className={classes.controls}>
        <input className={classes.inputUpload} id="file-uploader" type="file" multiple onChange={onSelectFile}/>
        <label className={classes.button} for="file-uploader">
          <i className={classes.materialIcons} role="presentation">attach_file</i>Add attachments
        </label>
        You can include multiple attachments (Max: 10.0 MB per issue)
      </div>
      {files.length === 0 ? null :
        (<ul>
          {
            files?.map((f, i) => (
              <li>
                {f.name}
                <Button onClick={onRemoveFile(i)}> remove</Button>
              </li>
            ))
          }
        </ul>)
      }
    </>
  );
}
