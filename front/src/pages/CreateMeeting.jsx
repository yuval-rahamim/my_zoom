import React, { useState, useEffect, useContext } from 'react'; 
import { useNavigate } from 'react-router-dom';
import Swal from 'sweetalert2';
import './CreateMeeting.css';
import { AuthContext } from '../components/AuthContext';

const CreateMeeting = () => {
  const [sessionId, setSessionId] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const [user, setUser] = useState(null);

  // Function to generate a random Room ID
  const generateRoomID = (length = 10) => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    return Array.from({ length }, () => chars[Math.floor(Math.random() * chars.length)]).join('');
  };

  useEffect(() => {
    setSessionId(generateRoomID()); // Generate Room ID on component mount

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
          console.log(data.user);
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
  }, [isLoggedIn, loading, navigate, logout]);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(sessionId);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleGenerateNewID = () => {
    setSessionId(generateRoomID());
    setCopied(false); // Reset "Copied!" status
  };

  const handleCreateSession = async (e) => {
    e.preventDefault();

    if (!user) {
      setError("You must be logged in to create a session.");
      return;
    }

    try {
      const response = await fetch('http://localhost:3000/sessions/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Ensures cookies are sent with the request
        body: JSON.stringify({ name: sessionId }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create session: ${response.statusText}`);
      }

      const data = await response.json();
      Swal.fire({
        icon: 'success',
        title: 'Meeting Created!',
        text: 'Session created successfully!',
      });

      // Redirect to the meeting room
      navigate(`/m/${sessionId}`);
      
    } catch (error) {
      setError(error.message);
      console.error('Error creating session:', error);
    }
  };

  return (
    <div className="card">
      <form className="form" onSubmit={handleCreateSession}>
        <h2 className="center-text">Create Meeting</h2>
        
        <div className="form-group">
          <label htmlFor="roomId">Room ID:</label>
          <div className="copy-container">
            <input
              type="text"
              id="roomId"
              value={sessionId}
              readOnly
              className="room-input"
            />
            <button type="button" onClick={copyToClipboard} className="copy-btn">
              {copied ? "Copied!" : "Copy"}
            </button>
          </div>
        </div>

        <button type="button" onClick={handleGenerateNewID} className="generate-btn">
          Generate New ID
        </button>

        {error && <p className="error">{error}</p>}

        <button type="submit" className="btn">Create Room</button>
      </form>
    </div>
  );
};

export default CreateMeeting;
