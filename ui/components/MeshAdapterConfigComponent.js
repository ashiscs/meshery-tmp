import React from 'react';
import PropTypes from 'prop-types';
import Button from '@material-ui/core/Button';
import { withStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import { NoSsr,  Chip, IconButton } from '@material-ui/core';
import dataFetch from '../lib/data-fetch';
import blue from '@material-ui/core/colors/blue';
import { updateAdaptersInfo, updateProgress } from '../lib/store';
import {connect} from "react-redux";
import { bindActionCreators } from 'redux';
import { withRouter } from 'next/router';
import CreatableSelect from 'react-select/lib/Creatable';
import ReactSelectWrapper from './ReactSelectWrapper';
import { withSnackbar } from 'notistack';
import CloseIcon from '@material-ui/icons/Close';


const styles = theme => ({
  root: {
    padding: theme.spacing(5),
  },
  buttons: {
    display: 'flex',
    justifyContent: 'flex-end',
  },
  button: {
    marginTop: theme.spacing(3),
    marginLeft: theme.spacing(1),
  },
  margin: {
    margin: theme.spacing(1),
  },
  alreadyConfigured: {
    textAlign: 'center',
    padding: theme.spacing(20),
  },
  colorSwitchBase: {
    color: blue[300],
    '&$colorChecked': {
      color: blue[500],
      '& + $colorBar': {
        backgroundColor: blue[500],
      },
    },
  },
  colorBar: {},
  colorChecked: {},
  fileLabel: {
    width: '100%',
  },
  fileLabelText: {
  },
  inClusterLabel: {
    paddingRight: theme.spacing(2),
  },
  alignCenter: {
    textAlign: 'center',
  },
  alignRight: {
    textAlign: 'right',
    marginBottom: theme.spacing(2),
  },
  fileInputStyle: {
    opacity: '0.01',
  },
  icon: {
    width: theme.spacing(2.5),
  },
  istioIcon: {
    width: theme.spacing(1.5),
  }
});

class MeshAdapterConfigComponent extends React.Component {

  constructor(props) {
    super(props);
    const {meshAdapters} = props;
    this.state = {
        meshAdapters,
        availableAdapters: [],
        ts: new Date(),
        meshLocationURLError: false,
      };
  }

  static getDerivedStateFromProps(props, state){
    const { meshAdapters, meshAdaptersts } = props;
    // if(meshAdapters.sort().join(',') !== state.meshAdapters.sort().join(',')){
    if(meshAdaptersts > state.ts) {
      return {
        meshAdapters, ts: meshAdaptersts
      };
    }
    return {};
  }

  componentDidMount = () => {
    this.fetchAvailableAdapters();
  }

  fetchAvailableAdapters = () => {
    let self = this;
    this.props.updateProgress({showProgress: true});
    dataFetch('/api/mesh/adapters', { 
      credentials: 'same-origin',
      method: 'GET',
      credentials: 'include',
    }, result => {
      this.props.updateProgress({showProgress: false});
      if (typeof result !== 'undefined'){
        const options = result.map(res => {
          return {
            value: res,
            label: res,
          };
        });
        this.setState({availableAdapters: options});
      }
    }, self.handleError("Unable to fetch available adapters"));
  }

  handleChange = name => event => {
    if (name === 'meshLocationURL' && event.target.value !== '') {
        this.setState({meshLocationURLError: false})
    }
    this.setState({ [name]: event.target.value });
  };

  handleMeshLocURLChange = (newValue, actionMeta) => {
    // console.log(newValue);
    // console.log(`action: ${actionMeta.action}`);
    // console.groupEnd();
    if (typeof newValue !== 'undefined'){
      this.setState({meshLocationURL: newValue, meshLocationURLError: false});
    }
  };
  handleInputChange = (inputValue, actionMeta) => {
    // console.log(inputValue);
    // console.log(`action: ${actionMeta.action}`);
    // console.groupEnd();

    // TODO: try to submit it and get 
    // if (typeof inputValue !== 'undefined'){
    //   this.setState({meshLocationURL: inputValue});
    // }
  }

  handleSubmit = () => {
    const { meshLocationURL } = this.state;
    
    if (!meshLocationURL || !meshLocationURL.value || meshLocationURL.value === ''){
        this.setState({meshLocationURLError: true})
        return;
      }

    this.submitConfig();
  }

  submitConfig = () => {
    const { meshLocationURL } = this.state;
    
    const data = {meshLocationURL: meshLocationURL.value};

    const params = Object.keys(data).map((key) => {
      return encodeURIComponent(key) + '=' + encodeURIComponent(data[key]);
    }).join('&');

    this.props.updateProgress({showProgress: true});
    let self = this;
    dataFetch('/api/mesh/manage', { 
      credentials: 'same-origin',
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
      },
      body: params
    }, result => {
      self.props.updateProgress({showProgress: false});
      if (typeof result !== 'undefined'){
        self.setState({meshAdapters: result, meshLocationURL: ''});
        self.props.enqueueSnackbar('Adapter was successfully configured!', {
          variant: 'success',
          autoHideDuration: 2000,
          action: (key) => (
            <IconButton
                  key="close"
                  aria-label="Close"
                  color="inherit"
                  onClick={() => self.props.closeSnackbar(key) }
                >
                  <CloseIcon />
            </IconButton>
          ),
        });
        self.props.updateAdaptersInfo({meshAdapters: result});
        self.fetchAvailableAdapters();
      }
    }, self.handleError("Adapter was not configured due to an error"));
  }

  handleDelete = (adapterLoc) => () => {
    // const { meshAdapters } = this.state;
    this.props.updateProgress({showProgress: true});
    let self = this;
    dataFetch(`/api/mesh/manage?adapter=${encodeURIComponent(adapterLoc)}`, { 
      credentials: 'same-origin',
      method: 'DELETE',
      credentials: 'include',
    }, result => {
      this.props.updateProgress({showProgress: false});
      if (typeof result !== 'undefined'){
        this.setState({meshAdapters: result});
         this.props.enqueueSnackbar('Adapter was successfully removed!', {
          variant: 'success',
          autoHideDuration: 2000,
          action: (key) => (
            <IconButton
                  key="close"
                  aria-label="Close"
                  color="inherit"
                  onClick={() => self.props.closeSnackbar(key) }
                >
                  <CloseIcon />
            </IconButton>
          ),
        });
        this.props.updateAdaptersInfo({meshAdapters: result});
      }
    }, self.handleError("Adapter was not removed due to an error"));
  }

  handleClick = (adapterLoc) => () => {
    // const { meshAdapters } = this.state;
    this.props.updateProgress({showProgress: true});
    let self = this;
    dataFetch(`/api/mesh/adapter/ping?adapter=${encodeURIComponent(adapterLoc)}`, { 
      credentials: 'same-origin',
      credentials: 'include',
    }, result => {
      this.props.updateProgress({showProgress: false});
      if (typeof result !== 'undefined'){
        this.props.enqueueSnackbar('Adapter was successfully pinged!', {
          variant: 'success',
          autoHideDuration: 2000,
          action: (key) => (
            <IconButton
                  key="close"
                  aria-label="Close"
                  color="inherit"
                  onClick={() => self.props.closeSnackbar(key) }
                >
                  <CloseIcon />
            </IconButton>
          ),
        });
      }
    }, self.handleError("error"));
  }

  handleError = (msg) => (error) => {
    this.props.updateProgress({showProgress: false});
    const self = this;
    this.props.enqueueSnackbar(`${msg}: ${error}`, {
      variant: 'error',
      action: (key) => (
        <IconButton
              key="close"
              aria-label="Close"
              color="inherit"
              onClick={() => self.props.closeSnackbar(key) }
            >
              <CloseIcon />
        </IconButton>
      ),
      autoHideDuration: 8000,
    });
  }

  configureTemplate = () => {
    const { classes } = this.props;
    const { availableAdapters, meshAdapters, meshLocationURL, meshLocationURLError } = this.state;
    
    let showAdapters = '';
    const self = this;
    if (meshAdapters.length > 0) {
      showAdapters = (
        <div className={classes.alignRight}>
          {meshAdapters.map((adapter, ind) => {
            let image = "/static/img/meshery-logo.png";
            let logoIcon = (<img src={image} className={classes.icon} />);
            switch (adapter.name.toLowerCase()){
              case 'istio':
                image = "/static/img/istio-blue.svg";
                logoIcon = (<img src={image} className={classes.istioIcon} />);
                break;
              case 'linkerd':
                image = "/static/img/linkerd.svg";
                logoIcon = (<img src={image} className={classes.icon} />);
                break;
              case 'consul':
                image = "/static/img/consul.svg";
                logoIcon = (<img src={image} className={classes.icon} />);
		            break;
              case 'network service mesh':
                image = "/static/img/nsm.svg";
                logoIcon = (<img src={image} className={classes.icon} />);
                break;
              case 'octarine':
                image = "/static/img/octarine.svg";
                logoIcon = (<img src={image} className={classes.icon} />);
                break;                
              // default:
            } 
            
            return (
            <Chip 
            label={adapter.adapter_location}
            onDelete={self.handleDelete(adapter.adapter_location)} 
            onClick={self.handleClick(adapter.adapter_location)} 
            icon={logoIcon} 
            variant="outlined" />
          );
          })}
          
        </div>
      )
    }


      return (
    <NoSsr>
    <div className={classes.root}>
    
    {showAdapters}
    
    <Grid container spacing={1} alignItems="flex-end">
      <Grid item xs={12}>

        {/* <CreatableSelect
          isClearable
          onChange={this.handleMeshLocURLChange}
          onInputChange={this.handleInputChange}
          options={availableAdapters}
        /> */}

        <ReactSelectWrapper
          onChange={this.handleMeshLocURLChange}
          onInputChange={this.handleInputChange}
          options={availableAdapters}
          value={meshLocationURL}
          // placeholder={'Mesh Adapter URL'}
          label={'Mesh Adapter URL'}
          error={meshLocationURLError}
        />

        {/* <TextField
          required
          id="meshLocationURL"
          name="meshLocationURL"
          label="Mesh Adapter Location"
          type="url"
          fullWidth
          value={meshLocationURL}
          error={meshLocationURLError}
          margin="normal"
          variant="outlined"
          onChange={this.handleChange('meshLocationURL')}
        /> */}
      </Grid>
    </Grid>
    <React.Fragment>
      <div className={classes.buttons}>
        <Button
          type="submit"
          variant="contained"
          color="primary"
          size="large"
          onClick={this.handleSubmit}
          className={classes.button}
        >
         Submit
        </Button>
      </div>
    </React.Fragment>
    </div>
  
  {/* <LoadTestTimerDialog open={timerDialogOpen} 
    t={t}
    onClose={this.handleTimerDialogClose} 
    countDownComplete={this.handleTimerDialogClose} />

  <Typography variant="h6" gutterBottom className={classes.chartTitle}>
      Results
    </Typography>
  <MesheryChart data={result} />     */}

    </NoSsr>
  );
    }

  render() {
    const { reconfigureCluster } = this.state;
    // if (reconfigureCluster) {
    return this.configureTemplate();
    // }
    // return this.alreadyConfiguredTemplate();
  }
}

MeshAdapterConfigComponent.propTypes = {
  classes: PropTypes.object.isRequired,
};

const mapDispatchToProps = dispatch => {
    return {
        updateAdaptersInfo: bindActionCreators(updateAdaptersInfo, dispatch),
        updateProgress: bindActionCreators(updateProgress, dispatch),
    }
}
const mapStateToProps = state => {
    const meshAdapters = state.get("meshAdapters").toJS();
    const meshAdaptersts = state.get("meshAdaptersts");
    return {meshAdapters, meshAdaptersts};
}

export default withStyles(styles)(connect(
    mapStateToProps,
    mapDispatchToProps
  )(withRouter(withSnackbar(MeshAdapterConfigComponent))));
