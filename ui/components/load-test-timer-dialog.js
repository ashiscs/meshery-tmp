import React from 'react';
import { NoSsr, Popper, Paper } from '@material-ui/core';
import dynamic from 'next/dynamic'
const ReactCountdownClock = dynamic(() => import('react-countdown-clock'), {
  ssr: false
})

class LoadTestTimerDialog extends React.Component {

  render() {
      const {countDownComplete, t, container, open} = this.props;
      let tNum = 0, dur;
      try {
        tNum = parseInt(t.substring(0, t.length - 1))
      }catch(ex){
      }
      switch(t.substring(t.length - 1, t.length).toLowerCase()) {
        case 'h':
          dur = tNum * 60 * 60;
          break;
        case 'm':
          dur = tNum * 60;
          break;
        default:
          dur = tNum;
      }
    return (
      <NoSsr>
        {/* <Menu
          // id="simple-menu"
          anchorEl={container}
          open={open}
          // onClose={this.handleClose}
        > */}
        <div id="anc1"></div>
        <Popper open={open} anchorEl={() => document.querySelector('#anc1')}
           placement='bottom' 
            modifiers={{
              flip: {
                enabled: false,
              },
            }}>
                {/* <Paper> */}
                  {/* <ClickAwayListener onClickAway={this.handleClose}></ClickAwayListener> */}

          {/* <MenuItem onClick={this.handleClose}>Profile</MenuItem>
          <MenuItem onClick={this.handleClose}>My account</MenuItem>
          <MenuItem onClick={this.handleClose}>Logout</MenuItem> */}
          <ReactCountdownClock seconds={dur}
                        color="#667C89"
                        alpha={0.9}
                        size={400}
                        onComplete={countDownComplete} 
                        />
                        {/* </Paper> */}
                        </Popper>
        {/* </Menu> */}
        
        {/* <Dialog onClose={this.handleTimerDialogClose} 
            {...other} hideBackdrop={true} >
            <DialogContent>
                    
            </DialogContent>
        </Dialog> */}
      </NoSsr>
    );
  }
}

export default LoadTestTimerDialog;
