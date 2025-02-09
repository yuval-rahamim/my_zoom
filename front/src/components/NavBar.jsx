import React, { useEffect, useState } from 'react'; 
import { Link, useNavigate } from 'react-router-dom';
import './NavBar.css';
import logo from '../assets/logo.webp';

const Navbar = () => {
  const [error, setError] = useState(null);
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(() => localStorage.getItem('token'));
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [darkMode, setDarkMode] = useState(() => localStorage.getItem('theme') === 'dark');
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState(false);

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
        console.error('Error fetching user:', error);
        setError(error.message);
        setIsLoggedIn(false);
      }
    };

    fetchUser();
  },[token]);

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

      setUser(null);
      setIsLoggedIn(false);
      navigate('/');
      window.location.reload();
    } catch (error) {
      setError(error.message);
    }
  };

  return (
    <nav className={`navbar ${darkMode ? 'dark' : 'light'}`}>
      <div className="navbar-container">
        
        <Link className="navbar-brand" to="/">
          <img className="logo-img" src={logo} width="60" alt="logo"/>
        </Link>
        <Link to="/home" className="navbar-brand">zoom</Link>

        {/* Hamburger Icon */}
        <div className={`hamburger ${isMenuOpen ? 'active' : ''}`} onClick={() => setIsMenuOpen(!isMenuOpen)}>
          <div className="bar"></div>
          <div className="bar"></div>
          <div className="bar"></div>
        </div>

        {/* Navbar Links */}
        <ul className={`navbar-links ${isMenuOpen ? 'active' : ''}`}>
          {!isLoggedIn ? (
            <>
              <li><Link to="/login">Login</Link></li>
              <li><Link to="/signup">Sign Up</Link></li>
            </>
          ) : (
            <>
              <div className='user-show' >
                <li> <Link to="/edit" ><img src={user.ImgPath} alt="" /> </Link></li>
                <li> <Link to="/edit"><span id='user'>{user?.Name || 'User'}</span></Link></li>
              </div>
              <li>
                <button className="Logout" onClick={logout}>
                  <div className="sign">
                    <svg viewBox="0 0 512 512">
                      <path d="M377.9 105.9L500.7 228.7c7.2 7.2 11.3 17.1 11.3 27.3s-4.1 20.1-11.3 27.3L377.9 406.1c-6.4 6.4-15 9.9-24 9.9c-18.7 0-33.9-15.2-33.9-33.9l0-62.1-128 0c-17.7 0-32-14.3-32-32l0-64c0-17.7 14.3-32 32-32l128 0 0-62.1c0-18.7 15.2-33.9 33.9-33.9c9 0 17.6 3.6 24 9.9zM160 96L96 96c-17.7 0-32 14.3-32 32l0 256c0 17.7 14.3 32 32 32l64 0c17.7 0 32 14.3 32 32s-14.3 32-32 32l-64 0c-53 0-96-43-96-96L0 128C0 75 43 32 96 32l64 0c17.7 0 32 14.3 32 32s-14.3 32-32 32z"></path>
                    </svg>
                  </div>
                  <div className="text">Logout</div>
                </button>
              </li>
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
}

export default Navbar;
