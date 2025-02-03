// App.jsx
import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
// import NavigationBar from './components/NavBar';
import Home from './Home.jsx';
import Login from './Login.jsx';
import SignUp from './Signup.jsx';
import './App.css';

const App = () => {
  return ( 
      <Router>
        <div>
          {/* <NavigationBar /> */}
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
