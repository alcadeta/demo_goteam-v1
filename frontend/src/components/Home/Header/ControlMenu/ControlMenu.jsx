import React from 'react';
import PropTypes from 'prop-types';
import { Col } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faPlusCircle } from '@fortawesome/free-solid-svg-icons';

import { capFirstLetterOf } from '../../../../misc/util';

import './controlmenu.sass';

const ControlMenu = ({
  name, isToggled, toggle, icon,
}) => (
  <Col xs={4} className="ControlMenu">
    <button
      className="Button"
      onClick={toggle}
      aria-label={`${name} button`}
      type="button"
    >
      <FontAwesomeIcon icon={icon} />
      {name.toUpperCase()}
    </button>

    {isToggled && (
      <div className={`Dropdown ${capFirstLetterOf(name)}Dropdown`}>
        <button type="button">
          Control Item #1
        </button>
        <button type="button">
          Control Item #2
        </button>
        <button type="button">
          <FontAwesomeIcon icon={faPlusCircle} />
        </button>
      </div>
    )}
  </Col>
);

ControlMenu.propTypes = {
  name: PropTypes.string.isRequired,
  isToggled: PropTypes.bool.isRequired,
  toggle: PropTypes.func.isRequired,
  icon: PropTypes.string.isRequired,
};

export default ControlMenu;