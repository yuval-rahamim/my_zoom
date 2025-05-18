import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import * as faceapi from 'face-api.js';
import shaka from 'shaka-player';
import './Meeting.css';

const Meeting = () => {
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const [participants, setParticipants] = useState([]);
  const [name, setName] = useState('');
  const localVideoRef = useRef(null);
  const videoRefs = useRef({});
  const canvasRefs = useRef({});
  const { id } = useParams();
  const [userID, setUserID] = useState();
  const navigate = useNavigate();

  const initializedParticipants = useRef(new Set());

  const startFaceDetection = (videoElement, canvas) => {
    const displaySize = { width: videoElement.videoWidth, height: videoElement.videoHeight };
    faceapi.matchDimensions(canvas, displaySize);

    const interval = setInterval(async () => {
      if (videoElement.paused || videoElement.ended) return;
      const detections = await faceapi
        .detectAllFaces(videoElement, new faceapi.TinyFaceDetectorOptions())
        .withFaceLandmarks()
        .withFaceExpressions();
      const resized = faceapi.resizeResults(detections, displaySize);

      canvas.getContext('2d').clearRect(0, 0, canvas.width, canvas.height);
      faceapi.draw.drawDetections(canvas, resized);
      faceapi.draw.drawFaceExpressions(canvas, resized);
    }, 500);

    return interval;
  };

  const startMedia = async (userId) => {
    let mediaRecorder;
    let socket;
    let stream;

    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
      localVideoRef.current.srcObject = stream;

      localVideoRef.current.onloadedmetadata = () => {
        const canvas = document.getElementById('local-face-canvas');
        startFaceDetection(localVideoRef.current, canvas);
      };

      socket = new WebSocket(`wss://myzoom.co.il:8080/b?userID=${userId}`);

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
    const res = await fetch('https://myzoom.co.il:3000/users/cookie', { credentials: 'include' });
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

  async function waitForMPD(streamURL, maxRetries = 30, delay = 1000) {
    for (let i = 0; i < maxRetries; i++) {
      try {
        const res = await fetch(streamURL, { method: 'HEAD' });
        if (res.ok) return true;
      } catch (_) {}
      await new Promise((res) => setTimeout(res, delay));
    }
    return false;
  }

  const initializeShakaPlayer = async (videoElement, url) => {
    const player = new shaka.Player();
    console.log(shaka.Player.version)
    await player.attach(videoElement);    

    player.configure({
      streaming: {
        bufferingGoal: 15,
        rebufferingGoal: 5,
        lowLatencyMode: true,
        alwaysStreamText: true
      },
      abr: {
        enabled: true // Automatically chooses best quality
      }
    });
    

    player.addEventListener('error', (e) => {
      console.error('Shaka Player error', e.detail);
    });

    try {
      await player.load(url);
  
      // Seek to live edge after stream is loaded
      // const seekRange = player.seekRange();
      // const desiredTime = seekRange.start + 20; // Seek to 5 seconds after the start of the seekable range
      // player.getMediaElement().currentTime = desiredTime;
      const seekRange = player.seekRange();
      player.getMediaElement().currentTime = seekRange.end - 5; // Seek to 5 seconds before live edge
      
    } catch (err) {
      console.error('Error loading stream:', err);
    }
  };

  const fetchParticipants = async () => {
    const res = await fetch(`https://myzoom.co.il:3000/sessions/${id}`, { credentials: 'include' });
    if (!res.ok) {
      navigate('/home');
      return;
    }

    const data = await res.json();
    setParticipants(data.participants);

    data.participants.forEach(async (p) => {
      if (p.streamURL && !initializedParticipants.current.has(p.id)) {
        const videoElement = videoRefs.current[p.id];
        if (videoElement) {
          const mpdExists = await waitForMPD(p.streamURL);
          if (mpdExists) {
            await initializeShakaPlayer(videoElement, p.streamURL);
            initializedParticipants.current.add(p.id);

            videoElement.onloadedmetadata = () => {
              const canvas = canvasRefs.current[p.id];
              if (canvas) {
                startFaceDetection(videoElement, canvas);
              }
            };
          }
        }
      }
    });
  };

  useEffect(() => {
    const loadModels = async () => {
      await faceapi.nets.tinyFaceDetector.loadFromUri('/models');
      await faceapi.nets.faceLandmark68Net.loadFromUri('/models');
      await faceapi.nets.faceRecognitionNet.loadFromUri('/models');
      await faceapi.nets.faceExpressionNet.loadFromUri('/models');
    };
    loadModels();
  }, []);

  useEffect(() => {
    if (!loading && isLoggedIn) {
      fetchUser();
    }
  }, [isLoggedIn, loading]);

  useEffect(() => {
    if (isLoggedIn && id) {
      fetchParticipants();
    }
  }, [id, isLoggedIn]);

  useEffect(() => {
    const ws = new WebSocket(`wss://myzoom.co.il:3000/ws`);
    ws.onopen = () => console.log('WebSocket connected!');
    ws.onmessage = (event) => {
      const message = event.data;
      if (message.includes('has joined') || message.includes('has left') || message.includes('stream started')) {
        fetchParticipants();
      }
    };
    return () => ws.close();
  }, []);

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
          <div className="video-wrapper">
            <video ref={localVideoRef} autoPlay muted playsInline className="video-player" />
            <canvas id="local-face-canvas" className="overlay-canvas" />
          </div>
        </div>

        {/* Remote Participants */}
        {participants.map((p) => (
          <div key={p.id} className="video-card">
            <h4>{p.name}</h4>
            <div className="video-wrapper">
              <video
                ref={(el) => (videoRefs.current[p.id] = el)}
                autoPlay
                playsInline
                muted={false}
                className="video-player"
                controls={true}
              />
              {/* <canvas
                ref={(el) => (canvasRefs.current[p.id] = el)}
                className="overlay-canvas"
              /> */}
            </div>
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
