import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Swal from 'sweetalert2';
import { AuthContext } from '../components/AuthContext';
import * as dashjs from 'dashjs';

const Meeting = () => {
    const [videoFile, setVideoFile] = useState(null);
    const [videoSrc, setVideoSrc] = useState('');
    const [name, setName] = useState('');
    const [participants, setParticipants] = useState([]);
    const { isLoggedIn, logout, loading } = useContext(AuthContext);
    const navigate = useNavigate();
    const [uploading, setUploading] = useState(false);
    const videoRef = useRef(null); // Ref for the video element
    const { id } = useParams(); // Get the id from the URL

    useEffect(() => {
        const fetchUser = async () => {
            try {
                const response = await fetch('http://localhost:3000/users/cookie', {
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
                    setName(data.user.Name);
                } else {
                    throw new Error('User data not found.');
                }
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
                console.log(data.participants);
                setParticipants(data.participants);
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
    }, [isLoggedIn, loading, navigate, logout, id]);

    const handleChange = (event) => {
        const file = event.target.files[0];
        if (file && file.type.startsWith('video/')) {
            setVideoFile(file);
        } else {
            Swal.fire('Error', 'Please select a valid video file', 'error');
            event.target.value = null; // Clear the invalid selection
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
            console.log(data)
            if (data.stream_url) {
                Swal.fire('Success', 'Video uploaded successfully', 'success');
                // Set the stream URL for the video player
                setVideoSrc(data.stream_url);
                setVideoFile(null); // Clear the selected file
            } else {
                console.error('No stream URL returned from backend');
                Swal.fire('Error', 'Stream URL not provided by the server.', 'error');
            }
        } catch (error) {
            console.error('Error uploading video:', error);
            Swal.fire('Error', error.message || 'Failed to upload video', 'error');
        } finally {
            setUploading(false);
        }
    };

    useEffect(() => {
        if (videoSrc && videoRef.current) {
            const player = dashjs.MediaPlayer().create();
            player.initialize(videoRef.current, videoSrc, true);

            return () => {
                player.reset(); // Dispose of the player when the component unmounts
            };
        }
    }, [videoSrc]);

    return (
        <div className="home">
            <div className="card">
                <h2 title="Meeting id" className="center-text">{id}</h2>
                <div className="form-group">
                    <label htmlFor="name">Name:</label>
                    <input
                        type="text"
                        id="name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                    />
                </div>
                <div className="form-group">
                    <label>
                        Upload Video:
                        <input type="file" accept="video/mp4" onChange={handleChange} />
                    </label>
                </div>

                {videoSrc && (
                    <video id="video-player" controls width="100%" ref={videoRef}>
                        <source src={videoSrc} type="application/dash+xml" />
                        Your browser does not support the video tag.
                    </video>
                )}

                <button onClick={handleVideoUpload} className="btn" disabled={uploading}>
                    {uploading ? 'Uploading...' : 'Upload Video'}
                </button>
            </div>

            {/* Display Participants */}
            <div className="participants">
                {participants.map((participant, index) => (
                    <div key={index} className="participant-card">
                        <video controls width="100%">
                            <source src={participant.stream_url} type="application/dash+xml" />
                            Your browser does not support the video tag.
                        </video>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default Meeting;
