// Spotify PWA Mock Player Logic

let isPlaying = false;
let audioContext = null;
let oscillator = null;
let gainNode = null;
let playbackInterval = null;
let elapsedSeconds = 0;
const totalDuration = 180; // 3 minutes mock song

// Silent base64 audio to keep media session alive on chrome
const silentAudio = new Audio('data:audio/wav;base64,UklGRigAAABXQVZFZm10IBIAAAABAAEARKwAAIhYAQACABAAAABkYXRhAgAAAAEA');
silentAudio.loop = true;

// DOM Elements
const playPauseBtn = document.getElementById('playPauseBtn');
const progressFill = document.getElementById('progressFill');
const currTimeEl = document.getElementById('currTime');
const playbackStatusEl = document.getElementById('playbackStatus');
const visualizerBars = document.querySelectorAll('.bar');

// Initialize PWA Service Worker
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js')
    .then(() => console.log('Service Worker Registered'))
    .catch((err) => console.log('Service Worker Failed', err));
}

// Format seconds into MM:SS
function formatTime(secs) {
  const m = Math.floor(secs / 60);
  const s = Math.floor(secs % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}

// Start Web Audio Synth
function startSynth() {
  if (!audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)();
  }
  
  if (audioContext.state === 'suspended') {
    audioContext.resume();
  }

  // Create oscillator for a soft ambient tone
  oscillator = audioContext.createOscillator();
  gainNode = audioContext.createGain();

  oscillator.type = 'triangle';
  oscillator.frequency.setValueAtTime(220, audioContext.currentTime); // A3 note
  
  // Subtle vibrato
  const lfo = audioContext.createOscillator();
  const lfoGain = audioContext.createGain();
  lfo.frequency.value = 4;
  lfoGain.gain.value = 3;
  lfo.connect(lfoGain);
  lfoGain.connect(oscillator.frequency);
  lfo.start();

  gainNode.gain.setValueAtTime(0, audioContext.currentTime);
  // Smooth fade-in
  gainNode.gain.linearRampToValueAtTime(0.15, audioContext.currentTime + 0.5);

  oscillator.connect(gainNode);
  gainNode.connect(audioContext.destination);
  
  oscillator.start();
}

// Stop Web Audio Synth
function stopSynth() {
  if (gainNode) {
    try {
      gainNode.gain.setValueAtTime(gainNode.gain.value, audioContext.currentTime);
      gainNode.gain.linearRampToValueAtTime(0, audioContext.currentTime + 0.2);
      setTimeout(() => {
        if (oscillator) {
          oscillator.stop();
          oscillator.disconnect();
          oscillator = null;
        }
      }, 200);
    } catch(e) {
      console.log('Error stopping oscillator:', e);
    }
  }
}

// Animate Visualizer Bars
let visualizerAnimation = null;
function animateVisualizer() {
  if (!isPlaying) return;
  visualizerBars.forEach(bar => {
    const height = Math.random() * 35 + 5;
    bar.style.height = `${height}px`;
  });
  visualizerAnimation = setTimeout(animateVisualizer, 100);
}

// Update Playback State UI and Media Session
function updatePlaybackState() {
  if (isPlaying) {
    playPauseBtn.innerHTML = `<svg viewBox="0 0 24 24"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>`;
    playbackStatusEl.innerText = 'Playing';
    playbackStatusEl.className = 'status-value status-active';
    animateVisualizer();
    
    // Play silent audio to trigger OS Media Control / MPRIS
    silentAudio.play().catch(e => console.log('Auto-play blocked or audio error:', e));

    if ('mediaSession' in navigator) {
      navigator.mediaSession.playbackState = 'playing';
    }

    // Start progress timer
    clearInterval(playbackInterval);
    playbackInterval = setInterval(() => {
      if (elapsedSeconds < totalDuration) {
        elapsedSeconds += 1;
        currTimeEl.innerText = formatTime(elapsedSeconds);
        progressFill.style.width = `${(elapsedSeconds / totalDuration) * 100}%`;
      } else {
        pauseTrack();
        elapsedSeconds = 0;
        currTimeEl.innerText = '0:00';
        progressFill.style.width = '0%';
      }
    }, 1000);

  } else {
    playPauseBtn.innerHTML = `<svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>`;
    playbackStatusEl.innerText = 'Paused';
    playbackStatusEl.className = 'status-value status-paused';
    clearTimeout(visualizerAnimation);
    visualizerBars.forEach(bar => bar.style.height = '5px');
    
    silentAudio.pause();
    clearInterval(playbackInterval);

    if ('mediaSession' in navigator) {
      navigator.mediaSession.playbackState = 'paused';
    }
  }
}

function playTrack() {
  isPlaying = true;
  startSynth();
  updatePlaybackState();
}

function pauseTrack() {
  isPlaying = false;
  stopSynth();
  updatePlaybackState();
}

// Play/Pause Action
playPauseBtn.addEventListener('click', () => {
  if (isPlaying) {
    pauseTrack();
  } else {
    playTrack();
  }
});

// Setup Media Session API (Critical for MPRIS bridge)
if ('mediaSession' in navigator) {
  navigator.mediaSession.metadata = new MediaMetadata({
    title: 'PWA Mock Spotify Song',
    artist: 'Antigravity Artist',
    album: 'Google DeepMind Salat Break Integration',
    artwork: [
      { 
        src: 'https://i.scdn.co/image/ab67616d0000b273b1897d28c89b2f293cfd9431', 
        sizes: '512x512', 
        type: 'image/png' 
      }
    ]
  });

  navigator.mediaSession.setActionHandler('play', () => {
    console.log('[MediaSession] Play triggered');
    playTrack();
  });

  navigator.mediaSession.setActionHandler('pause', () => {
    console.log('[MediaSession] Pause triggered');
    pauseTrack();
  });
}
