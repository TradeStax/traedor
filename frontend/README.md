# Traedor Frontend

A React/Next.js frontend for the Traedor algorithmic trading backtesting system.

## Features

- **Run History**: View and search previous backtest runs
- **New Backtest**: Configure and start new backtesting runs
- **Run Details**: View detailed results including performance metrics, trades, and charts
- **Signal Management**: Create and manage custom signal definitions
- **Responsive Design**: Works on desktop and mobile devices

## Getting Started

### Prerequisites

- Node.js 18 or higher
- npm or yarn

### Installation

1. Install dependencies:
```bash
cd frontend
npm install
```

2. Configure environment variables:
```bash
# Create .env.local file
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

3. Start the development server:
```bash
npm run dev
```

4. Open [http://localhost:3000](http://localhost:3000) in your browser.

## Building for Production

```bash
npm run build
npm start
```

## Project Structure

```
src/
├── components/     # Reusable React components
├── pages/         # Next.js pages (routing)
├── lib/           # Utility functions and API client
├── types/         # TypeScript type definitions
├── hooks/         # Custom React hooks
└── styles/        # Global styles and Tailwind CSS
```

## API Integration

The frontend communicates with the Go backend API through the `/api/v1` proxy configured in `next.config.js`. This allows for seamless development without CORS issues.

## Key Components

- **Layout**: Common layout with navigation
- **PerformanceChart**: Chart.js integration for equity curves
- **Forms**: React Hook Form for backtest configuration

## Technologies

- **Next.js 14**: React framework with SSR/SSG
- **TypeScript**: Type safety
- **Tailwind CSS**: Utility-first CSS framework
- **React Query**: Data fetching and caching
- **Chart.js**: Performance visualization
- **React Hook Form**: Form handling