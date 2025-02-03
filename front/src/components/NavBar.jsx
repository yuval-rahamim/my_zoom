import React, { useEffect, useState } from 'react'; 
import { Link, useNavigate } from 'react-router-dom';
import './NavBar.css';
import logo from '../assets/react.svg';

const Navbar = () => {
  const [error, setError] = useState(null);
  const [user, setUser] = useState({});
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [darkMode, setDarkMode] = useState(() => {
      return localStorage.getItem('theme') === 'dark';
    });
  const navigate = useNavigate();

  useEffect(() => {
      document.body.className = darkMode ? 'dark-mode' : 'light-mode';
      localStorage.setItem('theme', darkMode ? 'dark' : 'light');
    }, [darkMode]);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('http://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
        //   navigate('/'); 
        //   window.location.reload();
        setIsLoggedIn(false);
          if (response.status === 401) {
            throw new Error('Unauthorized. Please log in again.');
          }
          throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();
        if (data.user) {
          setUser(data.user);
          setIsLoggedIn(true);
        } else {
          throw new Error('User data not found.');
        }
      } catch (error) {
        // navigate('/'); 
        // window.location.reload();
        setError(error.message);
      }
    };

    if (isLoggedIn) {
      fetchUser();
    }
  }, []);

  const logout = async (e) => {
    e.preventDefault();

    try {
      const response = await fetch('http://localhost:3000/users/logout', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      localStorage.removeItem('token');
      navigate('/');
      window.location.reload();
    } catch (error) {
      setError(error.message);
    }
  };

  return (
    <nav className="navbar">
      <div className={`navbar-container ${darkMode ? 'dark' : 'light'}`}>
        <Link to="/" className="navbar-brand"><img src={logo} width="60"/></Link>
        <Link to="/" className="navbar-brand">zoom</Link>
        <ul className="navbar-links">
          {!isLoggedIn ? (
            <>
              <li><Link to="/login">Login</Link></li>
              <li><Link to="/signup">Sign Up</Link></li>
            </>
          ) : (
            <>
              <li><a id='user'>Hello, {user.Name || 'User'}</a></li>
              <li><button id='logout' onClick={logout}>Logout</button></li>
            </>
          )}
        </ul>
            <div className="toggle-container">
                <label className="switch">
                <input type="checkbox" checked={darkMode} onChange={() => setDarkMode(!darkMode)} />
                <span className="slider"></span>
                </label>
            </div>
      </div>
    </nav>
  );
};


export default Navbar;
