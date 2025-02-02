// App.jsx
import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import NavigationBar from './components/NavBar';
import Home from './pages/Home';
import Login from './pages/Login';
import SignUp from './pages/SignUp';
import './App.css';

const App = () => {
  return ( 
      <Router>
        <div>
          <NavigationBar />
          <div className="container">
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<SignUp />} />
            </Routes>
          </div>
        </div>
      </Router>
  );
};

export default App;
