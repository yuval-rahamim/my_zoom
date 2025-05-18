import React, { useState, useEffect, useContext } from 'react'; 
import { useNavigate } from 'react-router-dom';
import Swal from 'sweetalert2';
import './CreateMeeting.css';
import { AuthContext } from '../components/AuthContext';
import './JoinMeeting.css';

const JoinMeeting = () => {
  const [sessionId, setSessionId] = useState("");
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const [user, setUser] = useState(null);

  useEffect(() => {
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
      const response = await fetch(`https://myzoom.co.il:3000/sessions/join`, {
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
          <div className="paste-container">
            <input
              type="text"
              id="roomId"
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              className="room-input"
              placeholder="Paste or enter session ID"
            />
            <button type="button" className="paste" onClick={pasteFromClipboard}>
              <span data-text-end="Copied!" data-text-initial="paste from clipboard!" className="tooltip"></span>
              <span>
                <svg fill="currentColor" height="200px" width="200px" version="1.1" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 502 502" xmlSpace="preserve">
                  <g id="SVGRepo_bgCarrier" stroke-width="0"></g>
                  <g id="SVGRepo_tracerCarrier" stroke-linecap="round" stroke-linejoin="round"></g>
                  <g id="SVGRepo_iconCarrier"> <g> <g> <g> 
                  <path d="M467.35,190.176l-70.468-70.468c-1.876-1.875-4.419-2.929-7.071-2.929h-23.089V49c0-5.523-4.478-10-10-10h-115v-2.41 c0-20.176-16.414-36.59-36.59-36.59h-11.819c-20.176,0-36.591,16.415-36.591,36.59V39h-115c-5.522,0-10,4.477-10,10v386 c0,5.523,4.478,10,10,10h146.386v47c0,5.523,4.478,10,10,10h262.171c5.522,0,10-4.477,10-10V197.247 C470.279,194.595,469.225,192.051,467.35,190.176z M399.811,150.921l36.326,36.326h-36.326V150.921z M144.721,59h47 c5.522,0,10-4.477,10-10s-4.478-10-10-10h-15v-2.41c0-9.148,7.442-16.59,16.591-16.59h11.819c9.147,0,16.59,7.442,16.59,16.59V49 c0,5.523,4.478,10,10,10h22v20h-109V59z M198.107,116.779c-5.522,0-10,4.477-10,10V425H51.721V59h73v30c0,5.523,4.478,10,10,10 h129c5.522,0,10-4.477,10-10V59h73v57.779H198.107z M450.278,482H208.107V136.779H379.81v60.468c0,5.523,4.478,10,10,10h60.468 V482z"></path> <path d="M243.949,253.468h125.402c5.522,0,10-4.477,10-10c0-5.523-4.478-10-10-10H243.949c-5.522,0-10,4.477-10,10 C233.949,248.991,238.427,253.468,243.949,253.468z"></path>
                  <path d="M414.437,283.478H243.949c-5.522,0-10,4.477-10,10s4.478,10,10,10h170.487c5.522,0,10-4.477,10-10 S419.959,283.478,414.437,283.478z"></path>
                  <path d="M414.437,333.487H243.949c-5.522,0-10,4.477-10,10s4.478,10,10,10h170.487c5.522,0,10-4.477,10-10 S419.959,333.487,414.437,333.487z"></path> 
                  <path d="M414.437,383.497H243.949c-5.522,0-10,4.477-10,10s4.478,10,10,10h170.487c5.522,0,10-4.477,10-10 S419.959,383.497,414.437,383.497z"></path> 
                  <path d="M397.767,253.468h16.67c5.522,0,10-4.477,10-10c0-5.523-4.478-10-10-10h-16.67c-5.522,0-10,4.477-10,10 C387.767,248.991,392.245,253.468,397.767,253.468z"></path>
                  </g> </g> </g> </g>
                </svg>
              </span>
            </button>
          </div>
        </div>

        {error && <p className="error">{error}</p>}

        <button type="submit" className="btn">Join Room</button>
      </form>
    </div>
  );
};

export default JoinMeeting;
