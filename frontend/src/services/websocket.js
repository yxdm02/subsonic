import { ref } from 'vue';

const socket = ref(null);
const isConnected = ref(false);
const messageHandlers = new Map();

const connect = () => {
  if (isConnected.value) return;

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  const wsUrl = `${protocol}//${host}/ws`;

  socket.value = new WebSocket(wsUrl);

  socket.value.onopen = () => {
    isConnected.value = true;
    console.log('WebSocket connected');
  };

  socket.value.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.type && messageHandlers.has(data.type)) {
        messageHandlers.get(data.type)(data.payload);
      }
    } catch (error) {
      console.error('Error parsing WebSocket message:', error);
    }
  };

  socket.value.onclose = () => {
    isConnected.value = false;
    console.log('WebSocket disconnected. Attempting to reconnect...');
    setTimeout(connect, 3000); // Reconnect after 3 seconds
  };

  socket.value.onerror = (error) => {
    console.error('WebSocket error:', error);
    socket.value.close();
  };
};

const sendMessage = (type, payload) => {
  if (!isConnected.value) {
    console.error('WebSocket is not connected.');
    return;
  }
  const message = JSON.stringify({ type, payload });
  console.log('Sending message:', message); // Log the message to the console
  socket.value.send(message);
};

const on = (messageType, handler) => {
  messageHandlers.set(messageType, handler);
};

export function useWebSocket() {
  if (!socket.value) {
    connect();
  }
  return {
    isConnected,
    sendMessage,
    on,
  };
}
