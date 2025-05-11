import React, { useContext,useEffect, useState } from 'react'; 
import { useNavigate } from 'react-router-dom';
import { AuthContext } from '../components/AuthContext';

function LandingPage() {
  const [error, setError] = useState(null);
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const navigate = useNavigate();
  const [darkMode] = useState(() => {
      return localStorage.getItem('theme') === 'dark';
    });

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('https://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          logout();
          if (response.status === 401) {
            throw new Error('Unauthorized. Please log in again.');
          }
          throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();
        if (data.user) { 
          console.log(data.user)
        } else {
          throw new Error('User data not found.');
        }
      } catch (error) {
        setError(error.message);
        console.error('Error fetching user:', error);
      }
    };

    if (loading) return; // Don't do anything while auth is still loading

    if (isLoggedIn){
      fetchUser();
    }

  }, [isLoggedIn, loading]); 

  return (
    <div className={`Home ${darkMode ? 'dark' : 'light'}`}>
        {isLoggedIn ? <h1>landing page</h1> : <h1>Not signed in</h1>}
        <button>press</button>
    </div>
  );
}

export default LandingPage;