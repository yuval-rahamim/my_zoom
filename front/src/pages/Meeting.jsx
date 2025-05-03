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

  useEffect(() => {
    initializedParticipants.current.clear(); // Reset on participant change
    let lastParticipantsLength = participants.length;

    const initializeStreams = () => {
      let anyNew = false;

      participants.forEach((p) => {
        

        try {
          if (p.streamURL && videoRefs.current[p.id]) {
            const player = dashjs.MediaPlayer().create();
            player.initialize(videoRefs.current[p.id], p.streamURL, true);

            player.on(dashjs.MediaPlayer.events.ERROR, (e) => {
              console.error(`DASH error for ${p.name}:`, e);
            });

            initializedParticipants.current.add(p.id);
            anyNew = true;
            console.log(`Initialized stream for ${p.name}`);
          }
        } catch (err) {
          console.error(`Error initializing player for ${p.name}:`, err);
        }
      });

      // Stop checking if the number of participants hasn't changed and all streams are initialized
      if (initializedParticipants.current.size === participants.length && participants.length === lastParticipantsLength) {
        console.log('All streams initialized. No changes detected.');
        clearInterval(interval); // Stop the interval once all streams are initialized and no new participants join
      } else if (participants.length !== lastParticipantsLength) {
        // If the participants list changes, reset and continue checking
        lastParticipantsLength = participants.length;
        console.log('Participant count changed. Rechecking streams...');
      }
    };

    const interval = setInterval(initializeStreams, 1000);

    // return () => {
    //   clearInterval(interval); // Cleanup the interval on unmount or participants change
    // };
  }, [participants]); // Re-run when the participants list changes

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
            <video ref={(el) => (videoRefs.current[p.id] = el)} autoPlay muted playsInline className="video-player" />
            {!p.streamURL && <p>Waiting for stream...</p>}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Meeting;
