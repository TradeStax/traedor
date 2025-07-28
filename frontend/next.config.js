/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  async rewrites() {
    // Use environment variable or default to backend service name
    const backendHost = process.env.BACKEND_HOST || 'backend';
    const backendPort = process.env.BACKEND_PORT || '8080';
    const apiUrl = `http://${backendHost}:${backendPort}/api/:path*`;
    
    console.log('API rewrite configured to:', apiUrl);
    
    return [
      {
        source: '/api/:path*',
        destination: apiUrl,
      },
    ];
  },
};

module.exports = nextConfig;