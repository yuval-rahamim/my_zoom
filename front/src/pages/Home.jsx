import React, { useEffect, useState,useContext } from 'react'; 
import { useNavigate } from 'react-router-dom';
import './Home.css';
import { AuthContext } from '../components/AuthContext';

function Home() {
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const [darkMode] = useState(() => {
    return localStorage.getItem('theme') === 'dark';
  });

  const [user, setUser] = useState(null);
  const { isLoggedIn, logout, loading } = useContext(AuthContext);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('https://myzoom.co.il:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          logout()
          if (response.status === 401) {
            navigate('/login')
            throw new Error('Unauthorized. Please log in again.');
          }
          navigate('/signup')
          throw new Error(`HTTP error! Status: ${response.status}`);
        }
        const data = await response.json();
        if (data.user) { 
          console.log(data.user)
          setUser(data.user);
        } else {
          throw new Error('User data not found.');
        }
      } catch (error) {
        setError(error.message);
        console.error('Error fetching user:', error);
      }
    };

    if (loading) return; // Don't do anything while auth is still loading

    if (!isLoggedIn) {
      navigate('/login'); 
    } else {
      fetchUser();
    }

  }, [isLoggedIn, loading]); 

  return (
    <div className={`Home ${darkMode ? 'dark' : 'light'}`}>
        <h1>My home screen</h1>
        <div className='button-wrap'>
          <button className='b' onClick={()=>{navigate('/joinmeeting')}}>Join</button>
          <button className='b' onClick={()=>{navigate('/createmeeting')}}>Create</button>
          <button className='b' onClick={()=>{navigate('/friends')}}>Friends</button>
          <button className='b' onClick={()=>{navigate('/edit')}}>Edit user</button>
        </div>
    </div>
  );
}

export default Home;