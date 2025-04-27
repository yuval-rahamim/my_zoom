import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import * as dashjs from 'dashjs';
import './Meeting.css';

const Meeting = () => {
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const [participants, setParticipants] = useState([]);
  const [name, setName] = useState('');
  const localVideoRef = useRef(null);
  const videoRefs = useRef({});
  const { id } = useParams();
  const [userID,setUserID] = useState()
  const navigate = useNavigate();

  
  // User auth and session
  useEffect(() => {
      // Handle access to camera and microphone
    let mediaRecorder;
    let socket;
    let stream;
  
    const startMedia = async () => {
      try {
        // 1. Get access to camera/mic
        stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
        localVideoRef.current.srcObject = stream;
  
        // 2. Connect to WebSocket server
        socket = new WebSocket(`ws://localhost:8080/b?userID=${userID}`);
  
        socket.onopen = () => {
          console.log('WebSocket connected!');
  
          // 3. Start recording after WebSocket is open
          mediaRecorder = new MediaRecorder(stream, { mimeType: 'video/webm;codecs=vp9,opus' });
  
          mediaRecorder.ondataavailable = (event) => {
            if (event.data && event.data.size > 0 && socket.readyState === WebSocket.OPEN) {
              socket.send(event.data);
            }
          };
  
          mediaRecorder.start(1000); // record and send every 1 second
        };
  
        socket.onerror = (error) => {
          console.error('WebSocket error:', error);
        };
  
      } catch (err) {
        console.error('Media error:', err);
        Swal.fire('Error', 'Cannot access camera or microphone', 'error');
      }
    };

    const fetchUser = async () => {
      const res = await fetch('http://localhost:3000/users/cookie', { credentials: 'include' });
      if (!res.ok) {
        logout();
        navigate(res.status === 401 ? '/login' : '/signup');
        return;
      }
      const data = await res.json();
      setName(data.user.Name);
      setUserID(data.user.ID);
    };

    const fetchParticipants = async () => {
      const res = await fetch(`http://localhost:3000/sessions/${id}`, { credentials: 'include' });
      if (!res.ok) {
        navigate('/home');
        return;
      }
      const data = await res.json();
      setParticipants(data.participants);
    };

    if (!loading && isLoggedIn) {
      fetchUser();
      fetchParticipants();
      startMedia();
    }
  }, [isLoggedIn, loading, navigate, id, logout]);

  // Initialize DASH players for participants
  useEffect(() => {
    participants.forEach((p, i) => {
      if (p.streamURL && videoRefs.current[p.id]) {
        const player = dashjs.MediaPlayer().create();
        player.initialize(videoRefs.current[p.id], p.streamURL, true);
      }
    });
  }, [participants]);

  return (
    <div className="meeting-container">
      <div className="top-bar">
        <h2>Meeting ID: {id}</h2>
        <h3>Welcome, {name}</h3>
      </div>

      <div className="videos-container">
        {/* Local Video */}
        <div className="video-card">
          <h4>{name}</h4>
          <video ref={localVideoRef} autoPlay muted playsInline className="video-player" />
        </div>

        {/* Participants */}
        {participants.map((p) => (
          <div key={p.id} className="video-card">
            <h4>{p.name}</h4>
            <video
              ref={(el) => (videoRefs.current[p.id] = el)}
              controls
              className="video-player"
            />
            {!p.streamURL && <p>Waiting for stream...</p>}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Meeting;
