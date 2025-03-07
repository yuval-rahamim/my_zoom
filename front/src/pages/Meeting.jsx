import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import * as dashjs from 'dashjs';
import './Meeting.css';

const Meeting = () => {
  const [videoFile, setVideoFile] = useState(null);
  const [name, setName] = useState('');
  const [participants, setParticipants] = useState([]);
  const { isLoggedIn, logout, loading } = useContext(AuthContext);
  const navigate = useNavigate();
  const [uploading, setUploading] = useState(false);
  const { id } = useParams();
  const videoRefs = useRef({});
  const [socket, setSocket] = useState(null);
  const [dashReady, setDashReady] = useState(false);
  const localVideoRef = useRef(null);

  useEffect(() => {
    if (!loading && !isLoggedIn) {
      navigate("/login");
    }

    // Capture the camera feed
    const startCamera = async () => {
      try {
        const devices = await navigator.mediaDevices.enumerateDevices();
        const cameras = devices.filter(device => device.kind === "videoinput");
    
        if (cameras.length > 0) {
          const stream = await navigator.mediaDevices.getUserMedia({
            video: { deviceId: cameras[0].deviceId },
          });
          if (localVideoRef.current) {
            localVideoRef.current.srcObject = stream;
          }
        } else {
          console.error("No camera devices found");
          Swal.fire("Error", "No camera devices found", "error");
        }
      } catch (error) {
        console.error("Error accessing camera:", error);
        Swal.fire("Error", "Cannot access camera", "error");
      }
    };    

    startCamera();

    return () => {
      if (localVideoRef.current?.srcObject) {
        const tracks = localVideoRef.current.srcObject.getTracks();
        tracks.forEach(track => track.stop()); // Stop the camera when component unmounts
      }
    };
  }, [isLoggedIn, loading, navigate]);

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('http://localhost:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          logout();
          navigate(response.status === 401 ? '/login' : '/signup');
          throw new Error('Unauthorized. Please log in again.');
        }

        const data = await response.json();
        if (data.user) setName(data.user.Name);
      } catch (error) {
        console.error('Error fetching user:', error);
      }
    };

    const fetchSessionDetails = async () => {
      try {
        const response = await fetch(`http://localhost:3000/sessions/${id}`, {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) {
          navigate('/home');
          throw new Error('You are not part of this session.');
        }

        const data = await response.json();
        if (data.participants) {
          console.log(data.participants)
          setParticipants(data.participants);
        }
      } catch (error) {
        console.error('Error fetching session details:', error);
      }
    };

    if (!loading && !isLoggedIn) {
      navigate('/login');
    } else if (isLoggedIn) {
      fetchUser();
      fetchSessionDetails();
    }

    const ws = new WebSocket('ws://localhost:3000/ws');
    ws.onopen = () => console.log('WebSocket connection established');
    ws.onmessage = (event) => {
      const message = event.data;
      console.log('Received message:', message);
      if (message.includes('has joined')) {
        Swal.fire('New Participant', message, 'info');
        fetchSessionDetails();
      } else if (message.includes('MPEG-TS conversion complete')) {
        setDashReady(true);
        Swal.fire('MPEG-TS Complete', 'Ready for MPEG-DASH conversion.', 'info');
      }else if (message.includes('MPEG-DASH Ready')) {
        Swal.fire('MPEG-DASH Ready', 'The video is ready to play.', 'success');
        initializePlayers();
      }
    };
    ws.onerror = (error) => console.log('WebSocket error:', error);
    ws.onclose = () => console.log('WebSocket connection closed');

    setSocket(ws);

    return () => {
      ws.close();
    };
  }, [isLoggedIn, loading, navigate, logout, id]);

  const handleChange = (event) => {
    const file = event.target.files[0];
    if (file && file.type.startsWith('video/')) {
      setVideoFile(file);
    } else {
      Swal.fire('Error', 'Please select a valid video file', 'error');
    }
  };

  const handleVideoUpload = async () => {
    if (!videoFile) {
      Swal.fire('Error', 'Please select a video file first', 'error');
      return;
    }

    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('video', videoFile);

      const response = await fetch('http://localhost:3000/video/upload', {
        method: 'POST',
        credentials: 'include',
        body: formData,
      });

      const data = await response.json();
      if (response.ok) {
        Swal.fire('Success', data.message, 'success');
      } else {
        throw new Error(data.message || 'Failed to upload video');
      }
    } catch (error) {
      Swal.fire('Error', error.message || 'Failed to upload video', 'error');
    } finally {
      setUploading(false);
    }
  };

  const triggerMPEGDASHConversion = async () => {
    // Check if MPEG-TS conversion is complete before triggering DASH conversion
    if (!dashReady) {
      Swal.fire('Error', 'The MPEG-TS conversion has not completed yet.', 'error');
      return;
    }
  
    try {
      // Call the back-end API to start the MPEG-DASH conversion
      const response = await fetch('http://localhost:3000/video/dashconvert', {
        method: 'POST',
        credentials: 'include',  // include credentials if using session or cookies
      });
  
      // If the response is OK, show success
      if (response.ok) {
        Swal.fire('Success', 'MPEG-DASH conversion started!', 'success');
        
        const data = await response.json();
        if (data) {
          console.log('data:', data); // Handle the stream URL or show it to the user
        }

  
      } else {
        // Handle errors from the back-end response
        const errorData = await response.json();
        Swal.fire('Error', errorData.message || 'Failed to start DASH conversion.', 'error');
      }
    } catch (error) {
      // Catch network or other unexpected errors
      Swal.fire('Error', error.message || 'Failed to start DASH conversion.', 'error');
    }
  };

  const initializePlayers = () => {
    participants.forEach((participant, index) => {
      if (participant.streamURL && videoRefs.current[index]) {
        const player = dashjs.MediaPlayer().create();
        player.initialize(videoRefs.current[index], participant.streamURL, true);
      }
    });
  };
  
  useEffect(() => {
    initializePlayers(); // Call inside useEffect when participants change
  }, [participants]);

  return (
    <div className="home">
      <div className="card">
        <h2 className="center-text">Meeting ID: {id}</h2>
        <div className="form-group">
          <label>Upload Video:
            <input type="file" accept="video/*" onChange={handleChange} />
          </label>
        </div>
        <button onClick={handleVideoUpload} className="btn" disabled={uploading}>
          {uploading ? 'Uploading...' : 'Upload Video'}
        </button>
        {dashReady && (
          <button onClick={triggerMPEGDASHConversion} className="btn">
            Start MPEG-DASH Conversion
          </button>
        )}
      </div>
        {/* Local Camera Feed */}
      <div className="camera-feed">
        <h3>Your Camera</h3>
        <video ref={localVideoRef} autoPlay playsInline width="100%"></video>
      </div>

      {participants.length > 0 && (
        <div className="participants">
          {participants.map((participant, index) => (
            <div key={participant.id} className="card">
              <h2 className="center-text">{participant.name}</h2>
              <video
                controls
                width="100%"
                ref={(el) => (videoRefs.current[index] = el)}
                onLoadedMetadata={() => initializeDashPlayer(index, participant.streamURL)}
              />
              {!participant.streamURL && <p>Stream URL not available</p>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Meeting;