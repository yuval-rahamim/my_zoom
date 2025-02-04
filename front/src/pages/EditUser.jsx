import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Swal from "sweetalert2";

const EditUser = () => {
   const [user, setUser] = useState(null);
   const [error, setError] = useState(null);
   const [updatedUser, setUpdatedUser] = useState({
    Name: '',
    ImgPath: ''
  });
   const navigate = useNavigate();
 
   const [darkMode] = useState(() => {
     return localStorage.getItem('theme') === 'dark';
   });
 
   useEffect(() => {
       const fetchUser = async () => {
         try {
           const response = await fetch('http://localhost:3000/users/cookie', {
             method: 'GET',
             credentials: 'include',
           });
   
           if (!response.ok) {
             if (response.status === 401) {
               navigate('/login')
               throw new Error('Unauthorized. Please log in again.');
             }
             navigate('/signup')
             throw new Error(`HTTP error! Status: ${response.status}`);
           }
   
           const data = await response.json();
           if (data.user) {
             setUser(data.user);
             setUpdatedUser({Name: data.user.Name, ImgPath: data.user.ImgPath})
           } else {
             throw new Error('User data not found.');
           }
         } catch (error) {
           console.error('Error fetching user:', error);
           setError(error.message);
         }
       };
   
       fetchUser();
     },[]);

     const handleImageUpload = (e) => {
        const file = e.target.files[0];
        if (!file) return;
      
        // Convert image to Base64
        const reader = new FileReader();
        reader.onloadend = () => {
          // Set the Base64 string for preview and for submission
          setUpdatedUser(prev => ({ ...prev, ImgPath: reader.result }));
        };
        reader.readAsDataURL(file); // Convert the image to Base64
      };

  return (
    <div className="home">
      {error && <p className="error">{error}</p>}
      <div className={`card ${darkMode ? 'dark' : 'light'}`}>
      <h2 className='center-text'>Edit User Page</h2>
        <div className="form-group">
            <label htmlFor="name">Name:</label>
            <input
                type="text"
                id="name"
                value={updatedUser.Name}
                onChange={(e) => setUpdatedUser(prev => ({ ...prev, Name: e.target.value }))}
                required
            />
        </div>
        <div className="form-group">
            <label>
              Upload Image:
              <input
                type="file"
                name="Image"
                accept="image/*"
                onChange={handleImageUpload}
              />
            </label>
            {updatedUser.ImgPath && (
              <div className="image-preview">
                <img src={updatedUser.ImgPath} alt="Product Preview" width="100" />
              </div>
            )}
          </div> 
      </div>
    </div>
  );
};

export default EditUser;
