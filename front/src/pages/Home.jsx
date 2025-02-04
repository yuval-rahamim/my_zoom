import React, { useEffect, useState } from 'react'; 
import { useNavigate } from 'react-router-dom';
import './Home.css';

function Home() {
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const [darkMode] = useState(() => {
    return localStorage.getItem('theme') === 'dark';
  });

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('http://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          if (response.status === 401) {
            navigate('/login')
            window.location.reload();
            throw new Error('Unauthorized. Please log in again.');
          }
          navigate('/signup')
          window.location.reload();
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

    fetchUser();
  }, []); 

  return (
    <div className={`Home ${darkMode ? 'dark' : 'light'}`}>
        <h1>My home screen</h1>
        <div className='button-wrap'>
          <button className='b' >Join</button>
          <button className='b' onClick={()=>{navigate('/createmeeting')}}>Create</button>
          <button className='b'>Friends</button>
          <button className='b'>Edit user</button>
        </div>
    </div>
  );
}

export default Home;