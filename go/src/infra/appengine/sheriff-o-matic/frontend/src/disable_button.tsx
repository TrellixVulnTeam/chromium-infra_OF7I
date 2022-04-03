import React from 'react';

import { Alert, Button } from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CheckIcon from '@mui/icons-material/Check';
import { render } from 'react-dom';
import { CacheProvider } from '@emotion/react';
import createCache, { EmotionCache } from '@emotion/cache';

interface DisableTestButtonProps {
  testName: string;
  bugs: Bug[];
}

interface Bug {
  id: string;
}

export const DisableTestButton = (props: DisableTestButtonProps) => {
  const [wasCopied, setWasCopied] = React.useState(false);
  const [error, setError] = React.useState('');
  const onClick = () => {
    let command = "tools/disable_tests/disable '" + props.testName + "'";
    if (props.bugs && props.bugs.length === 1) {
      // Add the bug ID if present. Only do it if there's exactly one. If there
      // are more than one we don't know which one to use.
      command += " -b " + props.bugs[0].id;
    }

    navigator.clipboard.writeText(command).catch(function (err) {
      console.log(err);
      setError(err);
    });

    setWasCopied(true);
    setTimeout(() => setWasCopied(false), 1500);
  }
  return <>
    <Button size="small" id="copy-disable-command-button"
      title="Copy a command to run from the root of a chromium/src checkout to disable this test."
      onClick={onClick} startIcon={wasCopied ? <CheckIcon /> : <ContentCopyIcon />}>
      {wasCopied ? 'Copied' : 'Disable'}
    </Button>
    {error ?
      <Alert severity='error' onClose={() => setError('')}>{error}</Alert> :
      null}
  </>
}

export class SomDisableButton extends HTMLElement {
  cache: EmotionCache;
  child: HTMLSpanElement;
  props: DisableTestButtonProps = {
    testName: '',
    bugs: [],
  };

  constructor() {
    super();
    const root = this.attachShadow({ mode: 'open' });
    const parent = document.createElement('span');
    this.child = document.createElement('span');
    root.appendChild(parent).appendChild(this.child);
    this.cache = createCache({
      key: 'som-disable-button',
      container: parent,
    });
  }
  connectedCallback() {
    this.render();
  }

  set testName(value: string) {
    this.props.testName = value;
    this.render();
  }

  set bugs(value: Bug[]) {
    this.props.bugs = value;
    this.render();
  }

  render() {
    if (!this.isConnected) {
      return;
    }
    render(
      <CacheProvider value={this.cache}>
        <DisableTestButton {...this.props} />
      </CacheProvider>,
      this.child
    );
  }
}

customElements.define('som-disable-button', SomDisableButton);