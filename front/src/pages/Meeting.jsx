import React, { useEffect, useState,useContext } from 'react';
import { useNavigate } from 'react-router-dom';
import Swal from "sweetalert2";
import { AuthContext } from '../components/AuthContext';

const Meeting = () => {

    const [error, setError] = useState(null);
    const [videoFile, setVideoFile] = useState(null);
    const [videoSrc, setVideoSrc] = useState("");
    const navigate = useNavigate();
    const [darkMode] = useState(() => localStorage.getItem('theme') === 'dark');

    const [user, setUser] = useState(null);
    const [name, setName] = useState("");
    const { isLoggedIn, logout, loading } = useContext(AuthContext);

    useEffect(() => {
        const fetchUser = async () => {
        try {
            const response = await fetch('http://localhost:3000/users/cookie', {
            method: 'GET',
            credentials: 'include',
            });

            if (!response.ok) {
            logout()
            if (response.status === 401) {
                navigate('/login')
                throw new Error('Unauthorized. Please log in again.');
            }
            navigate('/signup')
            throw new Error(`HTTP error! Status: ${response.status}`);
            }
            const data = await response.json();
            if (data.user) { 
            console.log(data.user)
            setName(data.user.Name);
            setUser(data.user);
            } else {
            throw new Error('User data not found.');
            }
        } catch (error) {
            setError(error.message);
            console.error('Error fetching user:', error);
        }
        };

        if (loading) return; // Don't do anything while auth is still loading

        if (!isLoggedIn) {
        navigate('/login'); 
        } else {
        fetchUser();
        }

    }, [isLoggedIn, loading]); 

    // Handle video file selection
    const handleChange = (event) => {
        const file = event.target.files[0];
        if (file) {
            setVideoFile(file); // Store file in state
            setVideoSrc(URL.createObjectURL(file)); // Generate preview URL
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
            formData.append('Name', name); // Include existing user data
            formData.append('ImgPath', user.ImgPath);

            const response = await fetch('http://localhost:3000/video/upload', {
                method: 'POST',
                credentials: 'include',
                body: formData, // Send FormData instead of JSON
            });

            if (!response.ok) {
                throw new Error('Failed to update user');
            }

            const data = await response.json();
            Swal.fire("Success", "Video uploaded successfully", "success");
    
            // Redirect to watch the stream
            setVideoSrc(data.stream_url);
        } catch (error) {
            setError(error.message);
            console.error('Error updating user:', error);
            Swal.fire("Error", "Failed to update user", "error");
        }
    };

    return (
        <div className="home">
            {error && <p className="error">{error}</p>}
            <div className={`card ${darkMode ? 'dark' : 'light'}`}>
                <h2 className='center-text'>Meeting</h2>
                <div className="form-group">
                    <label htmlFor="name">Name:</label>
                    <input
                        type="text"
                        id="name"
                        value={name}
                        onChange={(e) =>
                        setName(e.target.value)
                        }
                        required
                    />
                </div>
                <div className="form-group">
                    <label>
                        Upload Video:
                        <input type="file" accept="video/mp4" onChange={handleChange} />
                    </label>
                </div>

                {/* Show video preview */}
                {videoSrc && (
                    <video controls width="100%">
                        <source src={videoSrc} type="video/mp4" />
                        Your browser does not support the video tag.
                    </video>
                )}

                <button onClick={handleVideoUpload} className="btn">Update User</button>
            </div>
        </div>
    );
};

export default Meeting;
