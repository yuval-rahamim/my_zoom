import React, { useState, useEffect, useContext } from 'react'; 
import { useNavigate } from 'react-router-dom';
import Swal from 'sweetalert2';
import './CreateMeeting.css';
import { AuthContext } from '../components/AuthContext';

const JoinMeeting = () => {
  const [sessionId, setSessionId] = useState("");
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const [user, setUser] = useState(null);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('http://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          logout();
          if (response.status === 401) {
            navigate('/login');
            throw new Error('Unauthorized. Please log in again.');
          }
          navigate('/signup');
          throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();
        if (data.user) { 
          setUser(data.user);
        } else {
          throw new Error('User data not found.');
        }
      } catch (error) {
        setError(error.message);
        console.error('Error fetching user:', error);
      }
    };

    if (loading) return; 
    if (!isLoggedIn) {
      navigate('/login'); 
    } else {
      fetchUser();
    }
  }, [isLoggedIn, loading, navigate, logout]);

  const pasteFromClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText();
      setSessionId(text);
    } catch (err) {
      console.error('Failed to read clipboard: ', err);
    }
  };

  const handleJoinSession = async (e) => {
    e.preventDefault();
    
    if (!sessionId.trim()) {
      setError("Please enter a valid session ID.");
      return;
    }

    try {
      const response = await fetch(`http://localhost:3000/sessions/join`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ name: sessionId }),
      });

      if (!response.ok) {
        throw new Error(`Failed to join session: ${response.statusText}`);
      }

      Swal.fire({
        icon: 'success',
        title: 'Joined Successfully!',
        text: 'You have joined the meeting.',
      });

      navigate(`/m/${sessionId}`);
    } catch (error) {
      setError(error.message);
      console.error('Error joining session:', error);
    }
  };

  return (
    <div className="card">
      <form className="form" onSubmit={handleJoinSession}>
        <h2 className="center-text">Join Meeting</h2>
        
        <div className="form-group">
          <label htmlFor="roomId">Session ID:</label>
          <div className="copy-container">
            <input
              type="text"
              id="roomId"
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              className="room-input"
              placeholder="Paste or enter session ID"
            />
            <button type="button" onClick={pasteFromClipboard} className="paste-btn">Paste</button>
          </div>
        </div>

        {error && <p className="error">{error}</p>}

        <button type="submit" className="btn">Join Room</button>
      </form>
    </div>
  );
};

export default JoinMeeting;
