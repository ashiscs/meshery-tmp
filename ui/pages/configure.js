import { NoSsr } from "@material-ui/core";
import MesheryConfigSteps from "../components/MesheryConfigSteps";
import { updatepagepath } from "../lib/store";
import {connect} from "react-redux";
import { bindActionCreators } from 'redux'
import { getPath } from "../lib/path";

class Config extends React.Component {
  componentDidMount () {
    console.log(`path: ${getPath()}`);
    this.props.updatepagepath({path: getPath()});
  }

  render () {
    return (
      <NoSsr>
        <MesheryConfigSteps />
      </NoSsr>
    );
  }
}

const mapDispatchToProps = dispatch => {
  return {
    updatepagepath: bindActionCreators(updatepagepath, dispatch)
  }
}

export default connect(
    null,
    mapDispatchToProps
  )(Config);