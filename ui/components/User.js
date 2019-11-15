import IconButton from '@material-ui/core/IconButton';
import Avatar from '@material-ui/core/Avatar';
import {connect} from "react-redux";
import { bindActionCreators } from 'redux'
import { updateUser } from '../lib/store';
import { withStyles } from '@material-ui/core/styles';
import MenuList from '@material-ui/core/MenuList';
import Grow from '@material-ui/core/Grow';
import MenuItem from '@material-ui/core/MenuItem';
import Popper from '@material-ui/core/Popper';
import Paper from '@material-ui/core/Paper';
import ClickAwayListener from '@material-ui/core/ClickAwayListener';
import NoSsr from '@material-ui/core/NoSsr';
import dataFetch from '../lib/data-fetch';


const styles = theme => ({
  popover: {
    color: 'black',
  },
});

class User extends React.Component {

  state = {
    user: null,
    open: false,
  }

  handleToggle = () => {
    this.setState(state => ({ open: !state.open }));
  };

  handleClose = event => {
    if (this.anchorEl.contains(event.target)) {
      return;
    }
    this.setState({ open: false });
  };

  handleLogout = () => {
    window.location = "/logout";
  };

  componentDidMount() {
    // console.log("fetching user data");
    dataFetch('/api/user', { credentials: 'same-origin' }, user => {
      this.setState({user})
      this.props.updateUser({user})
    }, error => {
      return {
        error
      };
    });
  }

  render() {
    const {color, iconButtonClassName, avatarClassName, classes, ...other} = this.props;
    let avatar_url, user_id;
    if (this.state.user !== null){
      avatar_url = this.state.user.avatar_url;
      user_id = this.state.user.user_id;
    }
    const { open } = this.state;
    return (
      <div>
        <NoSsr>
          <IconButton color={color} className={iconButtonClassName} 
          buttonRef={node => {
            this.anchorEl = node;
          }}
          aria-owns={open ? 'menu-list-grow' : undefined}
          aria-haspopup="true"
          onClick={this.handleToggle}>
            <Avatar className={avatarClassName}  src={avatar_url} />
          </IconButton>
              <Popper open={open} anchorEl={this.anchorEl} transition disablePortal placement='top-end'>
                {({ TransitionProps, placement }) => (
                <Grow
                  {...TransitionProps}
                  id="menu-list-grow"
                  style={{ transformOrigin: placement === 'bottom' ? 'left top' : 'left bottom' }}
                >
                <Paper className={classes.popover}>
                  <ClickAwayListener onClickAway={this.handleClose}>
                    <MenuList>
                      <MenuItem onClick={this.handleLogout}>Logout</MenuItem>
                    </MenuList>
                  </ClickAwayListener>
                </Paper>
              </Grow>
            )}
          </Popper>
          </NoSsr>
    </div>
    )
  }
}

const mapDispatchToProps = dispatch => {
  return {
    updateUser: bindActionCreators(updateUser, dispatch)
  }
}

export default withStyles(styles)(connect(
  null,
  mapDispatchToProps
)(User));