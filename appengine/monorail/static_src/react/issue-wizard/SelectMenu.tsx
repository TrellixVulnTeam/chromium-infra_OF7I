// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {createTheme, Theme} from '@material-ui/core/styles';
import {makeStyles} from '@material-ui/styles';
import MenuItem from '@material-ui/core/MenuItem';
import TextField from '@material-ui/core/TextField';

const theme: Theme = createTheme();

const useStyles = makeStyles((theme: Theme) => ({
  container: {
    display: 'flex',
    flexWrap: 'wrap',
    maxWidth: '65%',
  },
  textField: {
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
  },
  menu: {
    width: '100%',
    minWidth: '300px',
  },
}), {defaultTheme: theme});

/**
 * Select menu component that is located on the landing step if the
 * Issue Wizard. The menu is used for the user to indicate the category
 * of their bug when filing an issue.
 *
 * @return ReactElement.
 */
type Props = {
  optionsList: string[] | null,
  selectedOption: string | null,
  setOption: Function,
};

export default function SelectMenu(props: Props) {
  const classes = useStyles();

  const {optionsList, selectedOption, setOption} = props;

  const handleChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    setOption(event.target.value as string);
  };

  return (
    <form className={classes.container} noValidate autoComplete="off">
      <TextField
        id="outlined-select-category"
        select
        label=''
        className={classes.textField}
        value={selectedOption}
        onChange={handleChange}
        InputLabelProps={{shrink: false}}
        SelectProps={{
          MenuProps: {
            className: classes.menu,
          },
        }}
        margin="normal"
        variant="outlined"
        fullWidth={true}
      >
      {
        optionsList?.map(option => (
          <MenuItem
            className={classes.menu}
            key={option}
            value={option}
            data-testid="select-menu-item"
          >
             {option}
          </MenuItem>))
      }
      </TextField>
    </form>
  );
}
