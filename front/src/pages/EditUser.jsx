import React, { useEffect, useState, useContext } from "react";
import { redirect, useNavigate, useParams } from "react-router-dom";
import Swal from "sweetalert2";
import { AuthContext } from "../components/AuthContext";

const EditUser = () => {
  const [currentUser, setCurrentUser] = useState(null); // Logged-in user
  const [updatedUser, setUpdatedUser] = useState({
    Name: "",
    ImgPath: "",
    Manager: false,
  });

  const [error, setError] = useState(null);
  const { isLoggedIn, loading, setUserUpdated } = useContext(AuthContext);
  const navigate = useNavigate();
  const { name } = useParams(); // Get the username from the URL
  const [darkMode] = useState(() => localStorage.getItem("theme") === "dark");

  useEffect(() => {
    const fetchUsers = async () => {
      try {
        // Fetch the logged-in user
        const currentUserResponse = await fetch(
          "https://localhost:3000/users/cookie",
          {
            method: "GET",
            credentials: "include",
          }
        );

        if (!currentUserResponse.ok) {
          throw new Error("Failed to fetch logged-in user");
        }

        const currentUserData = await currentUserResponse.json();
        if (!currentUserData.user) {
          throw new Error("Current user data not found.");
        }
        setCurrentUser(currentUserData.user);

        // If editing another user, fetch their data
        if (name) {
          if(!currentUserData.user.Manager){
            Swal.fire("Error","You don't have permission to update this user", "error");
            navigate("/edit");
          }
          if(name==currentUserData.user.Name){
            navigate("/edit");

          }
          const targetUserResponse = await fetch(
            `https://localhost:3000/users/${name}`,
            {
              method: "GET",
              credentials: "include",
            }
          );

          if (!targetUserResponse.ok) {
            throw new Error("Failed to fetch target user");
          }

          const targetUserData = await targetUserResponse.json();
          console.log(targetUserData);
          if (targetUserData.user) {
            setUpdatedUser({
              Name: targetUserData.user.Name,
              ImgPath: targetUserData.user.ImgPath,
              Manager: targetUserData.user.Manager,
             });
          }
          if (!targetUserData.user) {
            throw new Error("Target user data not found.");
          }
        }  else {
          setUpdatedUser({
            Name: currentUserData.user.Name,
            ImgPath: currentUserData.user.ImgPath,
            Manager: currentUserData.user.Manager,
           });
        }

      } catch (error) {
        setError(error.message);
        console.error("Error fetching users:", error);
      }
    };

    if (loading) return;
    if (!isLoggedIn) {
      navigate("/login");
    } else {
      fetchUsers();
    }
  }, [isLoggedIn, loading, name]);

  const handleUpdateUser = async () => {
    if (updatedUser.Name.length < 3) {
      Swal.fire("Error", "User name must be at least 3 characters long", "error");
      return;
    }
  
    try {
      const endpoint = `https://localhost:3000/users/update`; // Unified endpoint
  
      const response = await fetch(endpoint, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          Name: updatedUser.Name,
          ImgPath: updatedUser.ImgPath,
          Manager: updatedUser.Manager,
          userName: name || undefined, // Send userName if updating another user's details
        }),
      });
  
      if (!response.ok) {
        // Handle specific error codes from the backend
        if (response.status === 403) {
          throw new Error("You don't have permission to update this user");
        }
        throw new Error("Failed to update user");
      }
  
      Swal.fire("Success", "User updated successfully", "success").then(() => {
        setUserUpdated((prev) => !prev);
        navigate("/home");
      });
    } catch (error) {
      setError(error.message);
      console.error("Error updating user:", error);
      Swal.fire("Error", error.message, "error");
    }
  };
  

  const handleImageUpload = (e) => {
    const file = e.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onloadend = () => {
      setUpdatedUser((prev) => ({ ...prev, ImgPath: reader.result }));
    };
    reader.readAsDataURL(file);
  };

  return (
    <div className="home">
      {error && <p className="error">{error}</p>}
      <div className={`card ${darkMode ? "dark" : "light"}`}>
        <h2 className="center-text">Edit {name || "Your"} Profile</h2>
        <div className="form-group">
          <label htmlFor="name">Name:</label>
          <input
            type="text"
            id="name"
            value={updatedUser.Name}
            onChange={(e) =>
              setUpdatedUser((prev) => ({ ...prev, Name: e.target.value }))
            }
            required
          />
        </div>
        {name && currentUser?.Manager && (
          <div className="form-group">
            <div className="checkbox-wrapper">
              <label className="checkbox-container" htmlFor="WaitingRoom">
                <input
                  type="checkbox"
                  id="WaitingRoom"
                  className="checkbox-input"
                  checked={updatedUser.Manager}
                  onChange={(e) =>
                    setUpdatedUser((prev) => ({
                      ...prev,
                      Manager: e.target.checked,
                    }))
                  }
                />
                <span className="checkbox-box"></span>
              </label>
              <label htmlFor="WaitingRoom">Is Manager</label>
            </div>
          </div>
        )}
        <div className="form-group">
          <label>
            Upload Image:
            <input type="file" accept="image/*" onChange={handleImageUpload} />
          </label>
          {updatedUser.ImgPath && (
            <div className="image-preview">
              <img
                style={{ borderRadius: "50%" }}
                src={updatedUser.ImgPath}
                alt="Profile Preview"
                width="100"
              />
            </div>
          )}
        </div>
        <button onClick={handleUpdateUser} className="btn">
          Update User
        </button>
      </div>
    </div>
  );
};

export default EditUser;
