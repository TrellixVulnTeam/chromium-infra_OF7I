// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import { makeStyles } from '@material-ui/core/styles';
import { createMuiTheme, ThemeProvider } from '@material-ui/core/styles';
import CssBaseline from '@material-ui/core/CssBaseline';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';

import NavBar from './components/NavBar';
import NavPanel from './components/NavPanel';
import MetricsPage from './pages/MetricsPage';
import { blue, teal } from '@material-ui/core/colors';

// Make sure that the App root css is imported.
import {} from './App.module.css';

const useStyles = makeStyles((theme) => ({
  content: {
    paddingTop: theme.spacing(4),
    paddingBottom: theme.spacing(4),
  },
}));

const theme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      main: blue[700],
    },
    secondary: {
      main: teal[200],
    },
  },
});

const App: React.FC = () => {
  const classes = useStyles();
  const [open, setOpen] = React.useState(false);

  const toggleDrawer = () => {
    setOpen(!open);
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Router>
        <NavBar toggleNavPanel={toggleDrawer} />
        <NavPanel open={open} />
        <main className={classes.content}>
          <Switch>
            <Route exact path="/" component={MetricsPage} />
            <Route path="/:dataSource" component={MetricsPage} />
          </Switch>
        </main>
      </Router>
    </ThemeProvider>
  );
};

export default App;
