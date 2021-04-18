import React from 'react';
import PropTypes from 'prop-types';
import { Container, Row } from 'react-bootstrap';
import { faUsers } from '@fortawesome/free-solid-svg-icons';

import BoardsControls from './Controls/Boards/BoardsControls';
import TeamControlsToggler from './Controls/Team/TeamControlsToggler';
import HelpToggler from './HelpToggler/HelpToggler';
import window from '../../../misc/window';

import logo from '../home.svg';
import './header.sass';

const Header = ({ activeWindow, handleActivate }) => (
  <div id="Header">
    <div className="Logo">
      <img alt="logo" src={logo} />
    </div>
    <div className="ControlsWrapper">
      <Container>
        <Row className="ControlsRow">
          <TeamControlsToggler
            isActive={activeWindow === window.TEAM}
            handleActivate={handleActivate(window.TEAM)}
            handleCreate={handleActivate(window.INVITE_MEMBER)}
            handleDelete={handleActivate(window.DELETE_MEMBER)}
            icon={faUsers}
          />

          <BoardsControls
            isActive={activeWindow === window.BOARDS}
            handleActivate={handleActivate(window.BOARDS)}
            handleCreate={handleActivate(window.CREATE_BOARD)}
            handleDelete={handleActivate(window.DELETE_BOARD)}
            handleEdit={handleActivate(window.EDIT_BOARD)}
          />

          <HelpToggler toggle={handleActivate(window.HELP)} />
        </Row>
      </Container>
    </div>
  </div>
);

Header.propTypes = {
  activeWindow: PropTypes.number.isRequired,
  handleActivate: PropTypes.func.isRequired,
};

export default Header;
