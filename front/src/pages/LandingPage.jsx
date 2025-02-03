import React, { useEffect, useState } from 'react'; 
import { useNavigate } from 'react-router-dom';

function Home() {
  const [error, setError] = useState(null);
  const [signedIn, setSignedIn] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('http://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
            setSignedIn(false);
          if (response.status === 401) {
            throw new Error('Unauthorized. Please log in again.');
          }
          navigate('/signup')
          throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();
        if (data.user) { 
          console.log(data.user)
          setSignedIn(true);
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
    <div className="Home">
        {signedIn ? <h1>landing page</h1> : <h1>Not signed in</h1>}
        <button>press</button>
    </div>
  );
}

export default Home;