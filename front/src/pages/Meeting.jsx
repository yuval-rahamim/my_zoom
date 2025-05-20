import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import * as dashjs from 'dashjs';
import * as faceapi from 'face-api.js';
import './Meeting.css';

const Meeting = () => {
 const { isLoggedIn, logout, loading } = useContext(AuthContext);
 const [participants, setParticipants] = useState([]);
 const [name, setName] = useState('');
 const localVideoRef = useRef(null);
 const videoRefs = useRef({});

 const [numOfUsers,setNumOfUsers] = useState(0);
 const dashPlayers = useRef({}); 
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
   const detections = await faceapi.detectAllFaces(videoElement, new faceapi.TinyFaceDetectorOptions()).withFaceLandmarks().withFaceExpressions().withFaceDescriptors().withAgeAndGender();
   const resized = faceapi.resizeResults(detections, displaySize);
   detections.length > 0 ? setNumOfUsers(detections.length) : setNumOfUsers(0);
   canvas.getContext('2d').clearRect(0, 0, canvas.width, canvas.height);
   faceapi.draw.drawDetections(canvas, resized);
   faceapi.draw.drawFaceExpressions(canvas, resized)
   // faceapi.draw.drawFaceLandmarks(canvas, resized);
     //ask AI to guess the age and gender
   resized.forEach(face=>{
    const { age, gender } = face
    const genderText = `${gender}`
    const ageText = `${Math.round(age)} years`
    const textField = new faceapi.draw.DrawTextField([genderText,ageText],face.detection.box.topRight)
    textField.draw(canvas)
  })
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

   // Wait for video to load metadata before starting face detection
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

    mediaRecorder.start(500);
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

 async function waitForMPD(streamURL, maxRetries = 40, delay = 1000) {
  for (let i = 0; i < maxRetries; i++) {
   try {
    const res = await fetch(streamURL, { method: 'HEAD' });
    if (res.ok) return true;
   } catch (_) {}
   await new Promise(res => setTimeout(res, delay));
  }
  return false;
 }
 
 const fetchParticipants = async () => {
  const res = await fetch(`https://myzoom.co.il:3000/sessions/${id}`, { credentials: 'include' });
  if (!res.ok) {
   navigate('/home');
   return;
  }
  const data = await res.json();
  setParticipants(data.participants);
 
  data.participants.forEach(async (p) => {
   if (p.streamURL && !initializedParticipants.current.has(p.id) ) {
    const videoElement = videoRefs.current[p.id];
    if (videoElement) {
     // Wait for the MPD file to be available before initializing the player
     const mpdExists = await waitForMPD(p.streamURL);
     if (mpdExists) {
      // Each participant gets their own Dash.js player instance
      dashPlayers.current[p.id] = dashjs.MediaPlayer().create(); 
      console.log(dashPlayers.current[p.id]);
      dashPlayers.current[p.id].updateSettings({
       streaming: {
        liveUpdateTimeThresholdInMilliseconds:1000,
        retryIntervals: { MPD: 500 },
       },
      });
 
      dashPlayers.current[p.id].initialize(videoElement, p.streamURL, true);
      initializedParticipants.current.add(p.id);
      // player.current.on('error', (e) => {
      //  console.error(`DASH error for ${p.name}:`, e);
      // });
 
      dashPlayers.current[p.id].on(dashjs.MediaPlayer.events.STREAM_INITIALIZED, () => {
       
       // if (player.current.isDynamic()) {
        dashPlayers.current[p.id].seekToOriginalLive();
       // }
      });
      
      // Start face detection once metadata is loaded
      
      videoElement.onloadedmetadata = () => {
       dashPlayers.current[p.id].seekToOriginalLive();

       const canvas = canvasRefs.current[p.id];
       if (canvas) {
        startFaceDetection(videoElement, canvas);
       }
       
      };
     } else {
      // console.error(`MPD file not found for participant ${p.name}`);
     }
    }
   }
  });
 }; 

 // 1. Load models only once on mount
 useEffect(() => {
  const loadModels = async () => {
   await faceapi.nets.tinyFaceDetector.loadFromUri('/models');
   await faceapi.nets.faceLandmark68Net.loadFromUri('/models');
   await faceapi.nets.faceRecognitionNet.loadFromUri('/models');
   await faceapi.nets.faceExpressionNet.loadFromUri('/models');
   await faceapi.nets.ageGenderNet.loadFromUri('/models');
   await faceapi.nets.ssdMobilenetv1.loadFromUri('/models');
  };
  const seekLiveInterval = setInterval(() => {
    Object.entries(dashPlayers.current).forEach(([id, player]) => {
      try {
        if (player && typeof player.seekToOriginalLive === 'function') {
          console.log(`[${id}] Seeking to live`);
          player.seekToOriginalLive();
        }
      } catch (e) {
        console.error(`[${id}] Seek live error:`, e);
      }
    });
  }, 10000);
  
  const printInterval = setInterval(() => {
    Object.entries(dashPlayers.current).forEach(([id, player]) => {
      try {
        const latency = player.getCurrentLiveLatency?.();
        console.log(`[${id}] Live latency:`, latency);
      } catch (e) {
        console.error(`[${id}] Live latency error:`, e);
      }
    });
  }, 1000);  

  loadModels();
  return () => {

   clearTimeout(seekLiveInterval);
   clearTimeout(printInterval);
  }
 }, []);

 // 2. Handle auth and start media when isLoggedIn is true and loading is done
 useEffect(() => {
  if (!loading && isLoggedIn) {
   fetchUser();
  }
 }, [isLoggedIn, loading]);

 // 3. Fetch participants only when session id changes and user is logged in
 useEffect(() => {
  if (isLoggedIn && id) {
   fetchParticipants();
  }
 }, [id, isLoggedIn]);

 // 4. WebSocket listener for participant updates (once on mount)
 useEffect(() => {
  const ws = new WebSocket(`wss://myzoom.co.il:3000/ws`);
  ws.onopen = () => console.log('WebSocket connected!');
  ws.onmessage = (event) => {
   const message = event.data;
   if (message.includes('has joined') || message.includes('has left') || message.includes('stream started')) {
    fetchParticipants();
   }
  };

  return () => ws.close(); // Cleanup on unmount
 }, []);

 // Cleanup Dash.js players when component unmounts
 useEffect(() => {
  return () => {
   for (const playerId in dashPlayers.current) {
    if (dashPlayers.current[playerId]) {
     dashPlayers.current[playerId].destroy();
    }
   }
  };
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
     <h4>Number of users: {numOfUsers}</h4>
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
        className="video-player"
        controls={false}
       />
       <canvas
        ref={(el) => (canvasRefs.current[p.id] = el)}
        className="overlay-canvas"
       />
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