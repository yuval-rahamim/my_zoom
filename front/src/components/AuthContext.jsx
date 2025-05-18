import React, { createContext, useState, useEffect } from 'react';

export const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [loading, setLoading] = useState(true);
  const [userUpdated, setUserUpdated] = useState(false); //  Add a trigger for user updates

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const response = await fetch('https://myzoom.co.il:3000/users/cookie', {
          method: 'GET',
          credentials: 'include',
        });

        if (!response.ok) throw new Error('Unauthorized');

        setIsLoggedIn(true);
      } catch (error) {
        console.error('Error checking auth:', error);
        setIsLoggedIn(false);
      } finally {
        setLoading(false);
      }
    };

    checkAuth();
  }, []);

  const login = () => setIsLoggedIn(true);
  const logout = () => setIsLoggedIn(false);

  return (
    <AuthContext.Provider value={{ isLoggedIn, login, logout, loading, userUpdated, setUserUpdated }}>
      {children}
    </AuthContext.Provider>
  );
};
