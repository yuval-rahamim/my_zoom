import React, { useState, useEffect, useContext } from 'react';
import './Friends.css';
import { AuthContext } from '../components/AuthContext';
import Swal from 'sweetalert2';

function Friends() {
  const [friends, setFriends] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [newFriend, setNewFriend] = useState('');
  const [filteredFriends, setFilteredFriends] = useState([]);
  const { isLoggedIn, loading } = useContext(AuthContext);

  useEffect(() => {
    if (!friends) return;
    const filtered = friends.filter(friend =>
      friend.user.Name.toLowerCase().includes(searchTerm.toLowerCase())
    );
    setFilteredFriends(filtered);
  }, [searchTerm, friends]);

  useEffect(() => {
    if (!loading && isLoggedIn) {
      fetchFriends();
    }
  }, [isLoggedIn, loading]);

  const fetchFriends = async () => {
    try {
      const res = await fetch('https://myzoom.co.il:3000/friends/all', {
        credentials: 'include',
      });
      const data = await res.json();
      setFriends(data.friends);
      
      console.log('Fetched friends:', data.friends);
    } catch (err) {
      console.error('Error fetching friends:', err);
      Swal.fire('Error', 'Failed to fetch friends.', 'error');
    }
  };

  const addFriend = async () => {
    try {
      const res = await fetch('https://myzoom.co.il:3000/friends/add', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ friendName: newFriend }),
      });
      if (res.ok) {
        setNewFriend('');
        fetchFriends();
        Swal.fire('Success', 'Friend added successfully!', 'success');
      } else {
        const data = await res.json();
        Swal.fire('Error', data.message || 'Failed to add friend.', 'error');
      }
    } catch (err) {
      console.error('Error adding friend:', err);
      Swal.fire('Error', 'Something went wrong when adding the friend.', 'error');
    }
  };

  const deleteFriend = async (name) => {
    try {
      const res = await fetch('https://myzoom.co.il:3000/friends/delete', {
        method: 'DELETE',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ friendName: name }),
      });
      if (res.ok) {
        fetchFriends();
        Swal.fire('Removed', 'Friend deleted successfully.', 'success');
      } else {
        const errData = await res.json();
        Swal.fire('Error', errData.message || 'Failed to delete friend.', 'error');
      }
    } catch (err) {
      console.error('Error deleting friend:', err);
      Swal.fire('Error', 'Something went wrong when deleting the friend.', 'error');
    }
  };

  const acceptFriend = async (name) => {
    try {
      const res = await fetch('https://myzoom.co.il:3000/friends/accept', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name }),
      });
      if (res.ok) {
        fetchFriends();
        Swal.fire('Accepted', 'Friend request accepted.', 'success');
      } else {
        const errData = await res.json();
        Swal.fire('Error', errData.message || 'Failed to accept friend.', 'error');
      }
    } catch (err) {
      console.error('Error accepting friend:', err);
      Swal.fire('Error', 'Something went wrong when accepting the friend.', 'error');
    }
  }

  return (
    <div className="card">
      <h2>Your Friends</h2>

      <div className="friend-controls">
        <input
          type="text"
          placeholder="Search friends..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
        <input
          type="text"
          placeholder="Add friend by name"
          value={newFriend}
          onChange={(e) => setNewFriend(e.target.value)}
        />
        <button onClick={addFriend}>Add</button>
      </div>

      <ul className="friend-list">
        {filteredFriends.map((friend) => (
          <li key={friend.user.ID}>
            <span>{friend.user.Name}</span>
            {friend.accepted ? (
              <span className="accepted"> (Accepted)</span>
            ) : (<>
              <span className="pending"> (Pending)</span>
              {friend.thisUserNeedToAccept && (
                <button onClick={() => acceptFriend(friend.user.Name)}>accept</button>
              )}
              </>
            )}
            <button onClick={() => deleteFriend(friend.user.Name)}>Remove</button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default Friends;
