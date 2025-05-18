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
        const response = await fetch('https://myzoom.co.il:3000/users/cookie', {
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
      const response = await fetch('https://myzoom.co.il:3000/sessions/create', {
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
            <button type="button" className="copy" onClick={copyToClipboard}>
              <span data-text-end="Copied!" data-text-initial="Copy to clipboard" className="tooltip"></span>
              <span>
                <svg
                  style={{ enableBackground: 'new 0 0 512 512' }}
                  viewBox="0 0 6.35 6.35"
                  height="20"
                  width="20"
                  xmlns="http://www.w3.org/2000/svg"
                  className="clipboard"
                >
                  <g>
                    <path
                      fill="currentColor"
                      d="M2.43.265c-.3 0-.548.236-.573.53h-.328a.74.74 0 0 0-.735.734v3.822a.74.74 0 0 0 .735.734H4.82a.74.74 0 0 0 .735-.734V1.529a.74.74 0 0 0-.735-.735h-.328a.58.58 0 0 0-.573-.53zm0 .529h1.49c.032 0 .049.017.049.049v.431c0 .032-.017.049-.049.049H2.43c-.032 0-.05-.017-.05-.049V.843c0-.032.018-.05.05-.05zm-.901.53h.328c.026.292.274.528.573.528h1.49a.58.58 0 0 0 .573-.529h.328a.2.2 0 0 1 .206.206v3.822a.2.2 0 0 1-.206.205H1.53a.2.2 0 0 1-.206-.205V1.529a.2.2 0 0 1 .206-.206z"
                    ></path>
                  </g>
                </svg>
                <svg
                  style={{ enableBackground: 'new 0 0 512 512' }}
                  viewBox="0 0 24 24"
                  height="18"
                  width="18"
                  xmlns="http://www.w3.org/2000/svg"
                  className="checkmark"
                >
                  <g>
                    <path
                      fill="currentColor"
                      d="M9.707 19.121a.997.997 0 0 1-1.414 0l-5.646-5.647a1.5 1.5 0 0 1 0-2.121l.707-.707a1.5 1.5 0 0 1 2.121 0L9 14.171l9.525-9.525a1.5 1.5 0 0 1 2.121 0l.707.707a1.5 1.5 0 0 1 0 2.121z"
                    ></path>
                  </g>
                </svg>
              </span>
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
