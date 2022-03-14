// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import Button from '@material-ui/core/Button';
import styles from './AttachmentUploader.css';

type Props = {
  files: Array<File>,
  setFiles: Function,
  setSubmitEnable: Function,
}

const isSameFile = (a: File, b: File) => {
  // NOTE: This function could return a false positive if two files have the
  // exact same name, lastModified time, size, and type but different
  // content. This is extremely unlikely, however.
  return a.name === b.name && a.lastModified === b.lastModified &&
    a.size === b.size && a.type === b.type;
}

const getTotalSize = (files: Array<File>) => {
  let size = 0;
  files.forEach((f) => {
    size += f.size;
  });
  return size;
}

const MAX_SIZE = 10 * 1000 * 1000;
export default function AttachmentUploader(props: Props): React.ReactElement {

  const {files, setFiles, setSubmitEnable} = props;
  const [isOverSize, setIsOverSize] = React.useState(false);

  const onSelectFile = (event: {currentTarget: any;}) => {
    const input = event.currentTarget;
    if (!input.files || input.files.length === 0) {
      return;
    }

    const newFiles = [...input.files].filter((f1) => {
      const fileExist = files.some((f2) => isSameFile(f1, f2));
      return !fileExist;
    })

    const expendFiles = [...files].concat(newFiles);
    const filesSize = getTotalSize(expendFiles);
    setIsOverSize(filesSize > MAX_SIZE);
    setSubmitEnable(filesSize <= MAX_SIZE);
    setFiles(expendFiles);
  }

  const onRemoveFile = (index: number) => () => {
    let remainingFiles = [...files];
    remainingFiles.splice(index, 1);
    const filesSize = getTotalSize(remainingFiles);
    setIsOverSize(filesSize > MAX_SIZE);
    setSubmitEnable(filesSize <= MAX_SIZE);
    setFiles(remainingFiles);
  }
  return (
    <>
      <div className={styles.controls}>
        <input className={styles.inputUpload} id="file-uploader" type="file" multiple onChange={onSelectFile}/>
        <label className={styles.button} for="file-uploader">
          <i className={styles.materialIcons} role="presentation">attach_file</i>Add attachments
        </label>
        You can include multiple attachments (Max: 10.0 MB per issue)
      </div>
      {files.length === 0 ? null :
        (<ul>
          {
            files?.map((f, i) => (
              <li>
                {f.name}
                <Button onClick={onRemoveFile(i)}> X</Button>
              </li>
            ))
          }
        </ul>)
      }
      {isOverSize ? <div className={styles.error}>Warning: Attachments are too big !</div> : null}
    </>
  );
}
