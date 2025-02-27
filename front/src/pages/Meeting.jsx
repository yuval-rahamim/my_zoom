import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import './Meeting.css';
import * as dashjs from 'dashjs';

const Meeting = () => {
    const [videoFile, setVideoFile] = useState(null);
    const [videoSrc, setVideoSrc] = useState('');
    const [name, setName] = useState('');
    const [participants, setParticipants] = useState([]);
    const { isLoggedIn, logout, loading } = useContext(AuthContext);
    const navigate = useNavigate();
    const [uploading, setUploading] = useState(false);
    const { id } = useParams(); // Get meeting ID from URL
    const videoRefs = useRef([]); // Array of refs for each participant
    const [socket, setSocket] = useState(null); // WebSocket state

    // Fetch user and session details on mount
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
                if (data.participants != null) {
                    setParticipants(data.participants); // Ensure participants have valid video URLs
                }
            } catch (error) {
                console.error('Error fetching session details:', error);
            }
        };

        // Check login status and fetch details
        if (!loading && !isLoggedIn) {
            navigate('/login');
        } else if (isLoggedIn) {
            fetchUser();
            fetchSessionDetails();
        }
    }, [isLoggedIn, loading, navigate, logout, id]);

    // Set up WebSocket connection without session ID in URL
    useEffect(() => {
        const socket = new WebSocket("ws://localhost:3000/ws"); // WebSocket URL no longer contains sessionID
        socket.onopen = () => {
            console.log('WebSocket connection established');
        };

        socket.onmessage = (event) => {
            const message = event.data;
            console.log('Received message:', message);

            // Handle join message
            if (message.includes('has joined')) {
                Swal.fire('New Participant', message, 'info');
                // You can refresh the participants list or append new participant here
                fetchSessionDetails(); // Optionally fetch session details to get the latest participant list
            }
        };

        socket.onerror = (error) => {
            console.log('WebSocket error:', error);
        };

        socket.onclose = () => {
            console.log('WebSocket connection closed');
        };

        setSocket(socket);

        // Cleanup WebSocket connection when the component unmounts
        return () => {
            socket.close();
        };
    }, [id]);

    const handleChange = (event) => {
        const file = event.target.files[0];
        if (file && file.type.startsWith('video/')) {
            setVideoFile(file);
        } else {
            Swal.fire('Error', 'Please select a valid video file', 'error');
            event.target.value = null;
            setVideoFile(null);
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

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || 'Failed to upload video');
            }

            const data = await response.json();
            if (data.stream_url) {
                Swal.fire('Success', 'Video uploaded successfully', 'success');
                console.log(data);
                setVideoSrc(data.stream_url);
                setVideoFile(null);
            } else {
                Swal.fire('Error', 'Stream URL not provided by the server.', 'error');
            }
        } catch (error) {
            Swal.fire('Error', error.message || 'Failed to upload video', 'error');
        } finally {
            setUploading(false);
        }
    };

    // Initialize dash.js for each participant's video stream
    useEffect(() => {
        participants.forEach((participant, index) => {
            if (participant.streamURL && videoRefs.current[index]) {
                const player = dashjs.MediaPlayer().create();
                player.initialize(videoRefs.current[index], participant.streamURL, true);
                return () => player.reset(); // Clean up when unmounting
            }
        });
    }, [participants, videoSrc]);

    return (
        <div className="home">
            <div className="card">
                <h2 title="Meeting id" className="center-text">{id}</h2>
                <div className="form-group">
                    <label>
                        Upload Video:
                        <input type="file" accept="video/mp4" onChange={handleChange} />
                    </label>
                </div>
                <button onClick={handleVideoUpload} className="btn" disabled={uploading}>
                    {uploading ? 'Uploading...' : 'Upload Video'}
                </button>
            </div>

            {/* Display Participants */}
            {participants.length > 0 && (
                <div className="participants">
                    {participants.map((participant, index) => {
                        return (
                            <div key={participant.id} className="card">
                                <h2 title="user name" className="center-text">{participant.name}</h2>
                                <video controls width="100%" ref={(el) => (videoRefs.current[index] = el)}>
                                    <source src={participant.streamURL} type="application/dash+xml" />
                                    Your browser does not support the video tag.
                                </video>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
};

export default Meeting;
