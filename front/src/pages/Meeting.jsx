import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Swal from "sweetalert2";

const Meeting = () => {
    const [user, setUser] = useState(null);
    const [error, setError] = useState(null);
    const [videoFile, setVideoFile] = useState(null);
    const [videoSrc, setVideoSrc] = useState("");
    const navigate = useNavigate();
    const [darkMode] = useState(() => localStorage.getItem('theme') === 'dark');

    useEffect(() => {
        const fetchUser = async () => {
            try {
                const response = await fetch('http://localhost:3000/users/cookie', {
                    method: 'GET',
                    credentials: 'include',
                });

                if (!response.ok) {
                    if (response.status === 401) {
                        navigate('/login');
                        window.location.reload();
                        throw new Error('Unauthorized. Please log in again.');
                    }
                    navigate('/signup');
                    window.location.reload();
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }

                const data = await response.json();
                if (data.user) {
                    setUser(data.user);
                } else {
                    throw new Error('User data not found.');
                }
            } catch (error) {
                console.error('Error fetching user:', error);
                setError(error.message);
            }
        };

        fetchUser();
    }, [navigate]);

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
            formData.append('Name', user.Name); // Include existing user data
            formData.append('ImgPath', user.ImgPath);

            const response = await fetch('http://localhost:3000/users/update', {
                method: 'PUT',
                credentials: 'include',
                body: formData, // Send FormData instead of JSON
            });

            if (!response.ok) {
                throw new Error('Failed to update user');
            }

            Swal.fire("Success", "User updated successfully", "success").then(() => {
                navigate('/home'); // Redirect after success
                window.location.reload();
            });
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
