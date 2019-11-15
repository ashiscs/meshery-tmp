import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { NoSsr, Grid, ExpansionPanelDetails, Typography, Dialog, Button, DialogActions, DialogContent, DialogTitle } from '@material-ui/core';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import LazyLoad from 'react-lazyload';
import GrafanaDateRangePicker from './GrafanaDateRangePicker';
import { ExpansionPanel, ExpansionPanelSummary } from './ExpansionPanels';
import GrafanaCustomChart from './GrafanaCustomChart';

const grafanaStyles = theme => ({
    root: {
      width: '100%',
    },
    column: {
      flexBasis: '33.33%',
    },
    heading: {
      fontSize: theme.typography.pxToRem(15),
    },
    secondaryHeading: {
      fontSize: theme.typography.pxToRem(15),
      color: theme.palette.text.secondary,
    },
    dateRangePicker: {
      display: 'flex',
      justifyContent: 'flex-end',
      marginRight: theme.spacing(1),
      marginBottom: theme.spacing(2),
    },
  });

class GrafanaCustomCharts extends Component {
  constructor(props) {
    super(props);

    const newStartDate = new Date();
    newStartDate.setMinutes(newStartDate.getMinutes() - 5);
    const {startDate, from, endDate, to, liveTail} = props;
    this.state = {
      startDate: startDate && startDate !== null?startDate:newStartDate,
      from: from && from !== null?from:'now-5m',
      endDate: endDate && endDate !== null?endDate:new Date(),
      to: to && to !== null?to:'now',
      liveTail: liveTail && liveTail !== null?liveTail:true,
      refresh: '10s',

      chartDialogOpen: false,
      chartDialogPanelData: {},
      chartDialogPanel: {},
      chartDialogBoard: {},
    }

  }
    updateDateRange = (from, startDate, to, endDate, liveTail, refresh) => {
      this.setState({from, startDate, to, endDate, liveTail, refresh});
    }

    genRandomNumberForKey = () => {
      return Math.floor((Math.random() * 1000) + 1);
    }

    chartDialogClose() {
      var self = this;
      return () => {
        self.setState({chartDialogOpen: false});
      }
    }

    handleChartDialogOpen = (board, panel, data) => {
      this.setState({chartDialogOpen: true, chartDialogBoard: board, chartDialogPanel: panel, 
        chartDialogPanelData: data });
    }
    
    render() {
        const {from, startDate, to, endDate, liveTail, refresh, chartDialogOpen, chartDialogPanel, chartDialogBoard, 
          chartDialogPanelData} = this.state;
        const { classes, boardPanelConfigs, boardPanelData, testUUID } = this.props;
        let {grafanaURL, grafanaAPIKey, prometheusURL} = this.props;
        // we are now proxying. . .
        // if (grafanaURL && grafanaURL.endsWith('/')){
        //   grafanaURL = grafanaURL.substring(0, grafanaURL.length - 1);
        // }
        return (
              <NoSsr>
              <React.Fragment>
              <div className={classes.root}>
                {!(boardPanelData && boardPanelData !== null) && 
                <div className={classes.dateRangePicker}>
                  <GrafanaDateRangePicker from={from} startDate={startDate} to={to} endDate={endDate} liveTail={liveTail} 
                    refresh={refresh} updateDateRange={this.updateDateRange} />
                </div>
                }
                <Dialog
                  fullWidth={true}
                  maxWidth="md"
                  open={chartDialogOpen}
                  onClose={this.chartDialogClose()}
                  aria-labelledby="max-width-dialog-title"
                >
                  <DialogTitle id="max-width-dialog-title">{chartDialogPanel.title}</DialogTitle>
                  <DialogContent>
                    {!(chartDialogPanelData && chartDialogPanelData !== null && Object.keys(chartDialogPanelData).length > 0) && 
                      <div className={classes.dateRangePicker}>
                        <GrafanaDateRangePicker from={from} startDate={startDate} to={to} endDate={endDate} liveTail={liveTail} 
                          refresh={refresh} updateDateRange={this.updateDateRange} />
                      </div>
                    }
                    <GrafanaCustomChart
                        key={this.genRandomNumberForKey()}
                        board={chartDialogBoard}
                        panel={chartDialogPanel}
                        handleChartDialogOpen={this.handleChartDialogOpen}
                        grafanaURL={grafanaURL}
                        grafanaAPIKey={grafanaAPIKey}
                        prometheusURL={prometheusURL}
                        from={from} startDate={startDate} to={to} endDate={endDate} liveTail={liveTail} refresh={refresh}
                        templateVars={chartDialogBoard.templateVars}
                        updateDateRange={this.updateDateRange}
                        inDialog={true}
                        testUUID={testUUID}
                        panelData={chartDialogPanelData && chartDialogPanelData !== null?chartDialogPanelData:{}}
                      /> 
                  </DialogContent>
                  <DialogActions>
                    <Button onClick={this.chartDialogClose()} color="primary">
                      Close
                    </Button>
                  </DialogActions>
                </Dialog>


                {boardPanelConfigs.map((config, ind) => (
                  // <ExpansionPanel defaultExpanded={ind === 0?true:false}>
                  <ExpansionPanel square defaultExpanded={ind === 0?true:false}>
                    <ExpansionPanelSummary expandIcon={<ExpandMoreIcon />}>
                      <div className={classes.column}>
                      <Typography variant="subtitle1" gutterBottom>{config.board && config.board.title?config.board.title:''}</Typography>
                      </div>
                      <div className={classes.column}>
                        <Typography variant="subtitle2">{config.templateVars && config.templateVars.length > 0?'Template variables: '+config.templateVars.join(' '):''}</Typography>
                      </div>
                    </ExpansionPanelSummary>
                    <ExpansionPanelDetails>
                        <Grid container spacing={5}>
                          {config.panels.map(panel => {
                            // if(panel.type === 'graph'){
                            return (
                              <Grid item xs={12} sm={6}>
                                <GrafanaCustomChart
                                  key={this.genRandomNumberForKey()}
                                  board={config}
                                  panel={panel}
                                  handleChartDialogOpen={this.handleChartDialogOpen}
                                  grafanaURL={grafanaURL}
                                  grafanaAPIKey={grafanaAPIKey}
                                  prometheusURL={prometheusURL}
                                  from={from} startDate={startDate} to={to} endDate={endDate} liveTail={liveTail} refresh={refresh}
                                  templateVars={config.templateVars}
                                  updateDateRange={this.updateDateRange}
                                  inDialog={false}
                                  testUUID={testUUID}
                                  panelData={boardPanelData && boardPanelData !== null && boardPanelData[ind] && boardPanelData[ind] !== null?
                                      boardPanelData[ind]:{}}
                                /> 
                            </Grid>
                            );
                            // } else return '';
                          })}
                        </Grid>
                    </ExpansionPanelDetails>
                  </ExpansionPanel>
                ))}
              </div>
              </React.Fragment>
              </NoSsr>
            );
        }
}

GrafanaCustomCharts.propTypes = {
  classes: PropTypes.object.isRequired,
  // grafanaURL: PropTypes.string.isRequired,
  // grafanaAPIKey: PropTypes.string.isRequired,
  boardPanelConfigs: PropTypes.array.isRequired,
  // boardPanelData: 
};

export default withStyles(grafanaStyles)(GrafanaCustomCharts);