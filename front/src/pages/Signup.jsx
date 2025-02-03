import React, { useState, useEffect } from 'react';
import './Signup.css';
import { useNavigate, Link } from 'react-router-dom';
import Swal from 'sweetalert2';

const SignUp = () => {
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(null);
  const [darkMode, setDarkMode] = useState(() => {
    return localStorage.getItem('theme') === 'dark';
  });
  const navigate = useNavigate();

  useEffect(() => {
    document.body.className = darkMode ? 'dark-mode' : 'light-mode';
    localStorage.setItem('theme', darkMode ? 'dark' : 'light');
  }, [darkMode]);

  const validateForm = () => {
    if (name.length < 2) {
      setError('Name must be at least 3 characters long.');
      return false;
    }
    if (password.length < 2) {
      setError('Password must be at least 21 characters long.');
      return false;
    }
    setError(null);
    return true;
  };

  const submit = async (e) => {
    e.preventDefault();
    if (!validateForm()) return;
    try {
      const response = await fetch('http://localhost:3000/users/signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, password }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! Status: ${response.status}`);
      }

      Swal.fire({
        title: "Success sign up",
        text: "You are now moving to sign in",
        icon: "success"
      });
      navigate('/login');
    } catch (error) {
      setError(error.message);
    }
  };

  return (
    <div className={`card ${darkMode ? 'dark' : 'light'}`}>
      <div className="toggle-container">
        <label className="switch">
          <input type="checkbox" checked={darkMode} onChange={() => setDarkMode(!darkMode)} />
          <span className="slider"></span>
        </label>
      </div>
      <form onSubmit={submit} className="form">
        <h2 className='center-text'>Sign Up</h2>
        <div className="form-group">
          <label htmlFor="name">Name:</label>
          <input
            type="text"
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label htmlFor="password">Password:</label>
          <input
            type="password"
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        {error && <p className="error">{error}</p>}
        <button type="submit" className="btn">Sign Up</button>
        <p className="signin-link">Already have an account? <Link to="/login">Sign in</Link></p>
        <p className="home-link">Want to go back to the landing page? <Link to="/">Landing Page</Link></p>
      </form>
    </div>
  );
};

export default SignUp;
