import React, { useState } from 'react';
import { Form, Button } from 'react-bootstrap';
import axios from 'axios';

import FormGroup from '../../_shared/FormGroup/FormGroup';
import inputType from '../../../misc/inputType';
import validateRegister from './validateRegister';

import logo from './register.svg';
import './register.sass';

const Register = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirmation, setPasswordConfirmation] = useState('');

  const [errors, setErrors] = useState({
    username: '',
    password: '',
    passwordConfirmation: '',
  });

  const handleSubmit = (e) => {
    e.preventDefault();

    const validationErrors = validateRegister(
      username,
      password,
      passwordConfirmation,
    );

    setErrors(validationErrors);

    if (!errors.username && !errors.password && !errors.passwordConfirmation) {
      axios.post(`${process.env.REACT_APP_BACKEND_URL}/register/`, {
        username,
        password,
        password_confirmation: passwordConfirmation,
      }).then((res) => {
        localStorage.setItem('username', res.data.username);
        localStorage.setItem('auth-token', res.data.token);
        console.log(`username: ${localStorage.getItem('username')}`);
        console.log(`auth-token: ${localStorage.getItem('auth-token')}`);
      }).catch((err) => {
        console.log(`ERROR OBJECT: ${JSON.stringify(err)}`);
      });
    }
  };

  return (
    <div id="Register">
      <Form className="Form" onSubmit={handleSubmit}>
        <div className="HeaderWrapper">
          <img className="Header" alt="logo" src={logo} />
        </div>

        <FormGroup
          type={inputType.TEXT}
          label="username"
          value={username}
          setValue={setUsername}
          error={errors.username}
        />

        <FormGroup
          type={inputType.PASSWORD}
          label="password"
          value={password}
          setValue={setPassword}
          error={errors.password}
        />

        <FormGroup
          type={inputType.PASSWORD}
          label="password confirmation"
          value={passwordConfirmation}
          setValue={setPasswordConfirmation}
          error={errors.passwordConfirmation}
        />

        <div className="ButtonWrapper">
          <Button className="Button" value="GO!" type="submit">
            GO!
          </Button>
        </div>

        <div className="Redirect">
          <p>Already have an account?</p>
          <p>
            <a href="/login">Login here.</a>
          </p>
        </div>
      </Form>
    </div>
  );
};

export default Register;
