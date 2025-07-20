/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  async rewrites() {
    const apiUrl = process.env.NODE_ENV === 'production' 
      ? 'http://backend:8080/api/v1/:path*'
      : 'http://localhost:8080/api/v1/:path*';
    
    return [
      {
        source: '/api/v1/:path*',
        destination: apiUrl,
      },
    ];
  },
};

module.exports = nextConfig;