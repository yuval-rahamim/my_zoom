import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as faceapi from 'face-api.js';
import './Vod.css';

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
      const res = await fetch('/user/meetings', { credentials: 'include' });
      if (!res.ok) {
        navigate(res.status === 401 ? '/login' : '/signup');
        return;
      }
      const data = await res.json();
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

  return (
    <div className="vod-container">
      <h1 className="vod-title">VOD Library</h1>
      {meetings.map((meeting) => (
        <div key={meeting.id} className="vod-meeting">
          <h2 className="vod-meeting-id">{meeting.id}</h2>
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
                      src={participant.videoUrl}
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
