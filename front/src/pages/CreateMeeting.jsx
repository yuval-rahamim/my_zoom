import React, { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import Swal from 'sweetalert2';
import './CreateMeeting.css';

const CreateMeeting = () => {
//   const [name, setName] = useState('');
//   const [password, setPassword] = useState('');
  const [WaitingRoom, setIsWaitingRoom] = useState(false)
  const [user, setUser] = useState(null);
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
              throw new Error('Unauthorized. Please log in again.');
            }
            navigate('/signup')
            throw new Error(`HTTP error! Status: ${response.status}`);
          }
  
          const data = await response.json();
          if (data.user) {
            setUser(data.user);
          } else {
            throw new Error('User data not found.');
          }
        } catch (error) {
          console.error('Error fetching user:', error);
          setError(error.message);
        }
      };
  
      fetchUser();
    },[]);

  const validateForm = () => {
    // if (name.length < 2) {
    //   setError('Name must be at least 3 characters long.');
    //   return false;
    // }
    // if (password.length < 2) {
    //   setError('Password must be at least 2 characters long.');
    //   return false;
    // }
    // setError(null);
    return true;
  };

  const submit = async (e) => {
    // e.preventDefault();
    // if (!validateForm()) return;
    // try {
    //   const response = await fetch('http://localhost:3000/users/signup', {
    //     method: 'POST',
    //     headers: { 'Content-Type': 'application/json' },
    //     body: JSON.stringify({ name, password }),
    //   });

    //   if (!response.ok) {
    //     const errorData = await response.json();
    //     throw new Error(errorData.message || `HTTP error! Status: ${response.status}`);
    //   }

    //   Swal.fire({
    //     title: "Success sign up",
    //     text: "You are now moving to sign in",
    //     icon: "success"
    //   });
    //   navigate('/login');
    // } catch (error) {
    //   setError(error.message);
    // }
  };

  return (
    <div className={`card ${darkMode ? 'dark' : 'light'}`}>
      <form onSubmit={submit} className="form">
        <h2 className='center-text'>Create meeting</h2>
        <div className="form-group">
            <div className="checkbox-wrapper">
                <label className="checkbox-container">
                    <input type="checkbox" className="checkbox-input" value={WaitingRoom} onChange={(e) => setIsWaitingRoom(e.target.value)}/>
                    <span className="checkbox-box"></span>
                </label>
                <div className='check'>
                    <label htmlFor="WaitingRoom">Waiting Room:</label>
                    <a>Only users admitted by the host can join the meeting</a>
                </div>
            </div>
        </div> 
        {error && <p className="error">{error}</p>}
        <button type="submit" className="btn">Create Room</button>
      </form>
    </div>
  );
};

export default CreateMeeting;
