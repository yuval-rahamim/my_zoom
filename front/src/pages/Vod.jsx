import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as faceapi from 'face-api.js';
import './Vod.css';
import Swal from 'sweetalert2';

const Vod = () => {
  const [meetings, setMeetings] = useState([]);
  const videoRefs = useRef({});
  const canvasRefs = useRef({});
  const intervals = useRef({});
  const navigate = useNavigate();

  useEffect(() => {
    const loadModels = async () => {
      const MODEL_URL = '/models';
      await Promise.all([
        faceapi.nets.tinyFaceDetector.loadFromUri(MODEL_URL),
        faceapi.nets.faceLandmark68Net.loadFromUri(MODEL_URL),
        faceapi.nets.faceRecognitionNet.loadFromUri(MODEL_URL),
      ]);
    };

    loadModels();
    fetchMeetings();

    return () => {
      Object.values(intervals.current).forEach(clearInterval);
    };
  }, []);

  const fetchMeetings = async () => {
    try {
      const res = await fetch('https://myzoom.co.il:3000/user/meetings', { credentials: 'include' });
      if (!res.ok) {
        navigate(res.status === 401 ? '/login' : '/signup');
        return;
      }
      const data = await res.json();
      console.log('Fetched meetings:', data);
      setMeetings(data);
    } catch (err) {
      console.error('Failed to fetch meetings:', err);
    }
  };

  const handleVideoPlay = async (meetingId, participantId) => {
    const key = `${meetingId}-${participantId}`;
    const video = videoRefs.current[key];
    const canvas = canvasRefs.current[key];

    if (!video || !canvas) return;

    if (!video.videoWidth || !video.videoHeight) {
      setTimeout(() => handleVideoPlay(meetingId, participantId), 300);
      return;
    }

    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;

    const displaySize = {
      width: video.videoWidth,
      height: video.videoHeight,
    };
    faceapi.matchDimensions(canvas, displaySize);

    const id = setInterval(async () => {
      if (video.paused || video.ended) return;

      const detections = await faceapi
        .detectAllFaces(video, new faceapi.TinyFaceDetectorOptions())
        .withFaceLandmarks()
        .withFaceDescriptors();

      const resized = faceapi.resizeResults(detections, displaySize);
      const ctx = canvas.getContext('2d');
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      faceapi.draw.drawDetections(canvas, resized);
      faceapi.draw.drawFaceLandmarks(canvas, resized);
    }, 300);

    intervals.current[key] = id;

    video.onpause = () => clearInterval(id);
    video.onended = () => clearInterval(id);
  };

  const deletVideo = async (meetingId) => {
    console.log('Deleting video with ID:', meetingId);
    try {
      const res = await fetch(`https://myzoom.co.il:3000/user/meetings/delete`, {
        method: 'DELETE',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ meetingId }),
      });
  
      if (!res.ok) {
        console.error('Failed to delete video:', res.statusText);
        Swal.fire({
          icon: 'error',
          title: 'Error',
          text: `Failed to delete video: ${res.statusText}`,
        });
        return;
      }
  
      const data = await res.json();
      console.log('Deleted video:', data);

      Swal.fire({
        icon: 'success',
        title: 'Success',
        text: 'Video deleted successfully!',
      });
      // Refresh meetings list
      fetchMeetings();
    } catch (err) {
      console.error('Error deleting video:', err);
    }
  };
  

  return (
    <div className="vod-container">
      <h1 className="vod-title">VOD Library</h1>
      {meetings.map((meeting) => (
        <div key={meeting.id} className="vod-meeting">
          <h2 className="vod-meeting-id">{meeting.name}(ID: {meeting.id})</h2>
          <button onClick={() => deletVideo(meeting.id)} className="vod-delete-button">
          🗑️
          </button>
          <div className="vod-video-grid">
            {meeting.participants.map((participant) => {
              const key = `${meeting.id}-${participant.id}`;
              return (
                <div key={key} className="vod-video-container">
                  <div className="vod-participant-info">
                    <strong>{participant.name}</strong> (ID: {participant.id})
                  </div>
                  <div className="vod-video-wrapper">
                    <video
                      ref={(el) => (videoRefs.current[key] = el)}
                      src={"https://myzoom.co.il:3000"+participant.video_url}
                      controls
                      onPlay={() => handleVideoPlay(meeting.id, participant.id)}
                      className="vod-video"
                    />
                    <canvas
                      ref={(el) => (canvasRefs.current[key] = el)}
                      className="vod-canvas"
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
};

export default Vod;
