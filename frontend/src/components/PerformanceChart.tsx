import { useMemo } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions,
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { Trade } from '@/types';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
);

interface PerformanceChartProps {
  trades: Trade[];
  startingBalance: number;
}

export default function PerformanceChart({ trades, startingBalance }: PerformanceChartProps) {
  const chartData = useMemo(() => {
    if (!trades || trades.length === 0) {
      return {
        labels: [],
        datasets: [],
      };
    }

    // Calculate running balance
    let balance = startingBalance;
    const equityPoints = [{ time: 'Start', balance }];

    trades
      .filter(trade => trade.net_profit !== undefined)
      .sort((a, b) => new Date(a.open_time).getTime() - new Date(b.open_time).getTime())
      .forEach((trade) => {
        balance += trade.net_profit!;
        equityPoints.push({
          time: new Date(trade.open_time).toLocaleDateString(),
          balance,
        });
      });

    return {
      labels: equityPoints.map(point => point.time),
      datasets: [
        {
          label: 'Account Balance',
          data: equityPoints.map(point => point.balance),
          borderColor: 'rgb(59, 130, 246)',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          borderWidth: 2,
          fill: true,
          tension: 0.1,
        },
      ],
    };
  }, [trades, startingBalance]);

  const options: ChartOptions<'line'> = {
    responsive: true,
    plugins: {
      legend: {
        position: 'top' as const,
      },
      title: {
        display: false,
      },
    },
    scales: {
      y: {
        beginAtZero: false,
        ticks: {
          callback: function(value) {
            return '$' + Number(value).toLocaleString();
          },
        },
      },
    },
    interaction: {
      intersect: false,
      mode: 'index',
    },
  };

  if (!trades || trades.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No trade data available for chart
      </div>
    );
  }

  return (
    <div className="h-64">
      <Line data={chartData} options={options} />
    </div>
  );
}