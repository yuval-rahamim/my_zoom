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
             setUpdatedUser({ Name: data.user.Name, ImgPath: data.user.ImgPath });
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

   const handleUpdateUser = async () => {
     if (updatedUser.Name.length < 3) {
       Swal.fire("Error", "User name must be at least 3 characters long", "error");
       return;
     }

     try {
       const response = await fetch('http://localhost:3000/users/update', {
         method: 'PUT',
         headers: {
           'Content-Type': 'application/json',
         },
         credentials: 'include',
         body: JSON.stringify({ Name: updatedUser.Name, ImgPath: updatedUser.ImgPath, }),
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

   const handleImageUpload = (e) => {
     const file = e.target.files[0];
     if (!file) return;

     const reader = new FileReader();
     reader.onloadend = () => {
       setUpdatedUser(prev => ({ ...prev, ImgPath: reader.result }));
     };
     reader.readAsDataURL(file);
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
             <input type="file" accept="image/*" onChange={handleImageUpload} />
           </label>
           {updatedUser.ImgPath && (
             <div className="image-preview">
               <img style={{ borderRadius: '50%' }} src={updatedUser.ImgPath} alt="Profile Preview" width="100" />
             </div>
           )}
         </div>
         <button onClick={handleUpdateUser} className="btn">Update User</button>
       </div>
     </div>
   );
};

export default EditUser;
