import React from 'react';
import App, { Container } from 'next/app';
import Head from 'next/head';
import { MuiThemeProvider, createMuiTheme, withStyles } from '@material-ui/core/styles';
import CssBaseline from '@material-ui/core/CssBaseline';
import getPageContext from '../components/PageContext';
import Navigator from '../components/Navigator';
import Header from '../components/Header';
import PropTypes from 'prop-types';
import Hidden from '@material-ui/core/Hidden';
import Paper from '@material-ui/core/Paper';
import withRedux from "next-redux-wrapper";
import { makeStore, actionTypes } from '../lib/store';
import {Provider} from "react-redux";
import { fromJS } from 'immutable'
import { NoSsr, Typography } from '@material-ui/core';
import FavoriteIcon from '@material-ui/icons/Favorite';
import { SnackbarProvider } from 'notistack';
import { MuiPickersUtilsProvider } from '@material-ui/pickers';
import MomentUtils from '@date-io/moment';

// codemirror + js-yaml imports when added to a page was preventing to navigating to that page using nextjs 
// link clicks, hence attemtpting to add them here
import 'codemirror/lib/codemirror.css';
import 'codemirror/theme/material.css';
import 'codemirror/addon/lint/lint.css';

// import 'billboard.js/dist/theme/insight.min.css';
// import 'billboard.js/dist/theme/graph.min.css';
import 'billboard.js/dist/billboard.min.css';

import { blueGrey, grey } from '@material-ui/core/colors';
import MesheryProgressBar from '../components/MesheryProgressBar';
import dataFetch from '../lib/data-fetch';

if (typeof window !== 'undefined') { 
  require('codemirror/mode/yaml/yaml'); 
  require('codemirror/mode/javascript/javascript'); 
  require('codemirror/addon/lint/lint');
  require('codemirror/addon/lint/yaml-lint');
  require('codemirror/addon/lint/json-lint');
  if (typeof window.jsyaml === 'undefined'){
    window.jsyaml = require('js-yaml');
  }
  if (typeof window.jsonlint === 'undefined'){
    // jsonlint did not work well with codemirror json-lint. Hence, found an alternative (jsonlint-mod) based on https://github.com/scniro/react-codemirror2/issues/21
    window.jsonlint = require('jsonlint-mod'); 
  }
}

let theme = createMuiTheme({
    typography: {
      useNextVariants: true,
      h5: {
        fontWeight: 500,
        fontSize: 26,
        letterSpacing: 0.5,
      },
    },
    palette: {
      // primary: {
      //   light: '#cfd8dc',
      //   main: '#607d8b',
      //   dark: '#455a64',
      // },
      primary: blueGrey,
      secondary: {
        main: '#EE5351',
      },
    },
    shape: {
      borderRadius: 8,
    },
  });
  
  theme = {
    ...theme,
    overrides: {
      MuiDrawer: {
        paper: {
          backgroundColor: '#263238',
        },
      },
      MuiButton: {
        label: {
          textTransform: 'initial',
        },
        contained: {
          boxShadow: 'none',
          '&:active': {
            boxShadow: 'none',
          },
        },
      },
      MuiTabs: {
        root: {
          marginLeft: theme.spacing(1),
        },
        indicator: {
          height: 3,
          borderTopLeftRadius: 3,
          borderTopRightRadius: 3,
        },
      },
      MuiTab: {
        root: {
          textTransform: 'initial',
          margin: '0 16px',
          minWidth: 0,
          // [theme.breakpoints.up('md')]: {
          //   minWidth: 0,
          // },
        },
        labelContainer: {
          padding: 0,
          // [theme.breakpoints.up('md')]: {
          //   padding: 0,
          // },
        },
      },
      MuiIconButton: {
        root: {
          padding: theme.spacing(1),
        },
      },
      MuiTooltip: {
        tooltip: {
          borderRadius: 4,
        },
      },
      MuiDivider: {
        root: {
          backgroundColor: '#404854',
        },
      },
      MuiListItemText: {
        primary: {
          fontWeight: theme.typography.fontWeightMedium,
        },
      },
      MuiListItemIcon: {
        root: {
          color: 'inherit',
          marginRight: 0,
          '& svg': {
            fontSize: 20,
          },
        },
      },
      MuiAvatar: {
        root: {
          width: 32,
          height: 32,
        },
      },
    },
    props: {
      MuiTab: {
        disableRipple: true,
      },
    },
    mixins: {
      ...theme.mixins,
      toolbar: {
        minHeight: 48,
      },
    },
  };
  
  const drawerWidth = 256;

  const styles = {
    root: {
      display: 'flex',
      minHeight: '100vh',
    },
    drawer: {
      [theme.breakpoints.up('sm')]: {
        width: drawerWidth,
        flexShrink: 0,
      },
    },
    appContent: {
      flex: 1,
      display: 'flex',
      flexDirection: 'column',
    },
    mainContent: {
      flex: 1,
      padding: '48px 36px 24px',
      background: '#eaeff1',
    },
    paper: {
        maxWidth: '90%',
        margin: 'auto',
        overflow: 'hidden',

      },
      footer: {
        backgroundColor: theme.palette.background.paper,
        padding: theme.spacing(2),
        color: '#737373',
      },
      footerText: {
        display: 'inline',
        verticalAlign: 'middle',
      },
      footerIcon: {
        display: 'inline',
        verticalAlign: 'top',
      },
      extl5: {
        cursor: 'pointer',
      }, 
      icon: {
        fontSize: 20,
      },
  };


class MesheryApp extends App {
  constructor() {
    super();
    this.pageContext = getPageContext();

    this.state = {
      mobileOpen: false,
    };
  }

  handleDrawerToggle = () => {
    this.setState(state => ({ mobileOpen: !state.mobileOpen }));
  };

  handleL5CommunityClick = () => {
    if (typeof window !== 'undefined'){
      const w = window.open('https://layer5.io', '_blank');
      w.focus();
    }
  }

  async loadConfigFromServer() {
    const { store } = this.props;
    const self = this;
      dataFetch('/api/config/sync', { 
	      credentials: 'same-origin',
	      method: 'GET',
	      credentials: 'include',
	    }, result => {
      if (typeof result !== 'undefined'){
        if(result.k8sConfig){
          if(typeof result.k8sConfig.inClusterConfig === 'undefined'){
            result.k8sConfig.inClusterConfig = false;
          }
          if(typeof result.k8sConfig.k8sfile === 'undefined'){
            result.k8sConfig.k8sfile = '';
          }
          if(typeof result.k8sConfig.contextName === 'undefined'){
            result.k8sConfig.contextName = '';
          }
          if(typeof result.k8sConfig.clusterConfigured === 'undefined'){
            result.k8sConfig.clusterConfigured = false;
          }
          if(typeof result.k8sConfig.configuredServer === 'undefined'){
            result.k8sConfig.configuredServer = '';
          }
          store.dispatch({ type: actionTypes.UPDATE_CLUSTER_CONFIG, k8sConfig: result.k8sConfig });
        }
        if(result.meshAdapters && result.meshAdapters !== null && result.meshAdapters.length > 0) {
          store.dispatch({ type: actionTypes.UPDATE_ADAPTERS_INFO, meshAdapters: result.meshAdapters });
        }
        if(result.grafana){
          if(typeof result.grafana.grafanaURL === 'undefined'){
            result.grafana.grafanaURL = '';
          }
          if(typeof result.grafana.grafanaAPIKey === 'undefined'){
            result.grafana.grafanaAPIKey = '';
          }
          if(typeof result.grafana.grafanaBoardSearch === 'undefined'){
            result.grafana.grafanaBoardSearch = '';
          }
          if(typeof result.grafana.grafanaBoards === 'undefined'){
            result.grafana.grafanaBoards = [];
          }
          if(typeof result.grafana.selectedBoardsConfigs === 'undefined'){
            result.grafana.selectedBoardsConfigs = [];
          }
          store.dispatch({ type: actionTypes.UPDATE_GRAFANA_CONFIG, grafana: result.grafana });
        }
        if(result.prometheus){
          if(typeof result.prometheus.prometheusURL === 'undefined'){
            result.prometheus.prometheusURL = '';
          }
          if(typeof result.prometheus.selectedPrometheusBoardsConfigs === 'undefined'){
            result.prometheus.selectedPrometheusBoardsConfigs = [];
          }
          store.dispatch({ type: actionTypes.UPDATE_PROMETHEUS_CONFIG, prometheus: result.prometheus });
        }
      }
      }, error => {
        console.log(`there was an error fetching user config data: ${error}`);
      });
  }

  static async getInitialProps({Component, ctx}) {
        const pageProps = Component.getInitialProps ? await Component.getInitialProps(ctx) : {};
        return {pageProps};
    }

  componentDidMount(){
    this.loadConfigFromServer(); // this works, but sometimes other components which need data load faster than this data is obtained.
  }

  render() {
    const { Component, store, pageProps, classes } = this.props;
    return (
      <NoSsr>
      <Container>
            <Provider store={store}>
                <Head>
                <title>Meshery</title>
                </Head>
                <MuiThemeProvider theme={theme}>
                  <MuiPickersUtilsProvider utils={MomentUtils}>
                        <div className={classes.root}>
                            <CssBaseline />
                            <nav className={classes.drawer}>
                                <Hidden smUp implementation="js">
                                <Navigator
                                    PaperProps={{ style: { width: drawerWidth } }}
                                    variant="temporary"
                                    open={this.state.mobileOpen}
                                    onClose={this.handleDrawerToggle}
                                />
                                </Hidden>
                                <Hidden xsDown implementation="css">
                                <Navigator PaperProps={{ style: { width: drawerWidth } }} />
                                </Hidden>
                            </nav>
                            <div className={classes.appContent}>
                                <Header onDrawerToggle={this.handleDrawerToggle} />
                                <SnackbarProvider
                                      anchorOrigin={{
                                          vertical: 'top',
                                          horizontal: 'right',
                                      }}
                                      maxSnack={5}
                                  >
                                <MesheryProgressBar />
                                <main className={classes.mainContent}>
                                    <MuiPickersUtilsProvider utils={MomentUtils}>
                                      <Paper className={classes.paper}>
                                          <Component pageContext={this.pageContext} {...pageProps} />
                                      </Paper>
                                    </MuiPickersUtilsProvider>
                                </main>
                                </SnackbarProvider>
                              <footer className={classes.footer}>
                                <Typography variant="body2" align="center" color="textSecondary" component="p">
                                  <span className={classes.footerText}>Built with <FavoriteIcon className={classes.footerIcon} /> by the <span onClick={this.handleL5CommunityClick} className={classes.extl5}>Layer5 Community</span></span>
                                </Typography>
                              </footer>
                            </div>
                        </div>
                    </MuiPickersUtilsProvider>
                </MuiThemeProvider>
            </Provider>
      </Container>
      </NoSsr>
    );
  }
}

MesheryApp.propTypes = {
    classes: PropTypes.object.isRequired,
  };
  
export default withStyles(styles)(withRedux(makeStore, {
    serializeState: state => state.toJS(),
    deserializeState: state => fromJS(state)
  })(MesheryApp));