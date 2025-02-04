// App.jsx
import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import NavigationBar from './components/NavBar';
import LandingPage from './pages/LandingPage.jsx';
import Home from './pages/Home.jsx';
import Login from './pages/Login.jsx';
import SignUp from './pages/Signup.jsx';
import CreateMeeting from './pages/CreateMeeting.jsx'
import EditUser from './pages/EditUser.jsx'

import './styles/App.css';

const App = () => {
  return ( 
      <Router>
        <div>
          <NavigationBar />
          <div className="container">
            <Routes>
              <Route path="/" element={<LandingPage />} />
              <Route path="/home" element={<Home />} />
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<SignUp />} />
              <Route path="/createmeeting" element={<CreateMeeting />}></Route>
              <Route path="/edit" element={<EditUser />}></Route>
            </Routes>
          </div>
        </div>
      </Router>
  );
};

export default App;
