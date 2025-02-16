import React, { useState, useEffect, useContext } from 'react';
import { useNavigate } from 'react-router-dom';
import Swal from "sweetalert2";
import { AuthContext } from '../components/AuthContext';
import MediaPlayer from 'dashjs';

const Meeting = () => {
    const [videoFile, setVideoFile] = useState(null);
    const [videoSrc, setVideoSrc] = useState("");
    const [name, setName] = useState("");
    const { isLoggedIn, logout, loading } = useContext(AuthContext);
    const navigate = useNavigate();

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

        if (!loading && !isLoggedIn) {
            navigate('/login');
        } else if (isLoggedIn) {
            fetchUser();
        }
    }, [isLoggedIn, loading, navigate, logout]);

    // Handle video file selection
    const handleChange = (event) => {
        const file = event.target.files[0];
        if (file) {
            setVideoFile(file);
        }
    };

    // Upload video file
    const handleVideoUpload = async () => {
        if (!videoFile) {
            Swal.fire("Error", "Please select a video file first", "error");
            return;
        }

        try {
            const formData = new FormData();
            formData.append('video', videoFile);
            formData.append('Name', name);

            const response = await fetch('http://localhost:3000/video/upload', {
                method: 'POST',
                credentials: 'include',
                body: formData,
            });

            if (!response.ok) {
                throw new Error('Failed to upload video');
            }

            const data = await response.json();

            // Set video stream URL (DASH Manifest)
            if (data.stream_url) {
                Swal.fire("Success", "Video uploaded successfully", "success");
                setVideoSrc(data.stream_url);
            } else {
                console.error("No stream URL returned from backend");
            }
        } catch (error) {
            console.error('Error uploading video:', error);
            Swal.fire("Error", "Failed to upload video", "error");
        }
    };

    // Initialize the DASH player once the video source is available
    useEffect(() => {
        if (videoSrc) {
            const video = document.getElementById('video-player');
            if (video) {
                const player = MediaPlayer.create();
                player.initialize(video, videoSrc, true); // Initialize with the .mpd stream URL
            }
        }
    }, [videoSrc]);

    return (
        <div className="home">
            <div className="card">
                <h2 className='center-text'>Meeting</h2>
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

                {/* Show DASH Video Player */}
                {videoSrc && (
                    <video id="video-player" controls width="100%">
                        <source src={videoSrc} type="application/dash+xml" />
                        Your browser does not support the video tag.
                    </video>
                )}

                <button onClick={handleVideoUpload} className="btn">Upload Video</button>
            </div>
        </div>
    );
};

export default Meeting;
