import React, { useEffect } from 'react';

function Notification({ notification, onClear }) {
  useEffect(() => {
    if (notification) {
      const timer = setTimeout(onClear, 4000);
      return () => clearTimeout(timer);
    }
  }, [notification, onClear]);

  if (!notification) return null;

  return (
    <div className={`notification ${notification.type}`}>
      {notification.message}
    </div>
  );
}

export default Notification;
