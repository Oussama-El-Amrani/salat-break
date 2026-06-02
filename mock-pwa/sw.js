// Service Worker for Spotify PWA Mock
const CACHE_NAME = 'spotify-mock-cache-v1';
const ASSETS = [
  '/',
  '/index.html',
  '/app.js',
  '/app.css',
  '/manifest.json'
];

self.addEventListener('install', (e) => {
  e.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(ASSETS);
    })
  );
});

self.addEventListener('fetch', (e) => {
  e.respondWith(
    caches.match(e.request).then((response) => {
      return response || fetch(e.request);
    })
  );
});
