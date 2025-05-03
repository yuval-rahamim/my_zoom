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
  const [userID, setUserID] = useState();
  const navigate = useNavigate();

  const initializedParticipants = useRef(new Set());

  const startMedia = async (userId) => {
    let mediaRecorder;
    let socket;
    let stream;

    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
      localVideoRef.current.srcObject = stream;

      socket = new WebSocket(`ws://localhost:8080/b?userID=${userId}`);

      socket.onopen = () => {
        console.log('WebSocket connected!');

        mediaRecorder = new MediaRecorder(stream, { mimeType: 'video/webm;codecs=vp9,opus' });

        mediaRecorder.ondataavailable = (event) => {
          if (event.data && event.data.size > 0 && socket.readyState === WebSocket.OPEN) {
            socket.send(event.data);
          }
        };

        mediaRecorder.start(1000);
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
    startMedia(data.user.ID);
  };

  const fetchParticipants = async () => {
    const res = await fetch(`http://localhost:3000/sessions/${id}`, { credentials: 'include' });
    if (!res.ok) {
      navigate('/home');
      return;
    }
    const data = await res.json();
    setParticipants(data.participants);
  
    data.participants.forEach((p) => {
      if (p.streamURL && !initializedParticipants.current.has(p.id)) {
        const videoElement = videoRefs.current[p.id];
        if (videoElement) {
          const player = dashjs.MediaPlayer().create();
          player.initialize(videoElement, p.streamURL, true);
          initializedParticipants.current.add(p.id);
          player.on('error', (e) => {
            console.error(`DASH error for ${p.name}:`, e);
          });
          
        }
      }
    });
  };  

  useEffect(() => {
    if (!loading && isLoggedIn) {
      fetchUser();
      fetchParticipants();
    }

    const ws = new WebSocket(`ws://localhost:3000/ws`);
    ws.onopen = () => console.log('WebSocket connected!');
    ws.onmessage = (event) => {
      const message = event.data;
      console.log('Received message:', message);
      if (message.includes('has joined')) {
        console.log('Participant joined:', message);
        fetchParticipants();
      } else if (message.includes('has left')) {
        console.log('Participant left:', message);
        fetchParticipants();
      }
    };
  }, [isLoggedIn, loading, navigate, id, logout]);
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
              autoPlay
              playsInline
              className="video-player"
            />
            {!p.streamURL ? (
              <p>Waiting for stream...</p>
            ) : (
              <p className="live-indicator">Live</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Meeting;
