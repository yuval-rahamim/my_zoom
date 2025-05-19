// App.jsx
import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import NavigationBar from './components/NavBar';
import LandingPage from './pages/LandingPage.jsx';
import Home from './pages/Home.jsx';
import Login from './pages/Login.jsx';
import SignUp from './pages/Signup.jsx';
import CreateMeeting from './pages/CreateMeeting.jsx'
import JoinMeeting from './pages/JoinMeeting.jsx'
import EditUser from './pages/EditUser.jsx'
import Meeting from './pages/Meeting.jsx'
import {AuthProvider} from './components/AuthContext'

import './styles/App.css';
import Friends from './pages/Friends.jsx';
import Vod from './pages/Vod.jsx';

const App = () => {
  return ( 
      <Router>
        <AuthProvider>
          <NavigationBar />
          <div className="container">
            <Routes>
              <Route path="/" element={<LandingPage />} />
              <Route path="/home" element={<Home />} />
              <Route path="/friends" element={<Friends />} />
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<SignUp />} />
              <Route path="/createmeeting" element={<CreateMeeting />}></Route>
              <Route path="/joinmeeting" element={<JoinMeeting />}></Route>
              <Route path="/edit/:name?" element={<EditUser />} />
              <Route path="/m/:id?" element={<Meeting />}></Route>
              <Route path="/vod" element={<Vod />}></Route>
            </Routes>
          </div>
        </AuthProvider>
      </Router>
  );
};

export default App;
