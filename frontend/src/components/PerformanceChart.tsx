import { useEffect, useRef, memo } from 'react';
import { createChart, ColorType, IChartApi, ISeriesApi, LineData } from 'lightweight-charts';
import { Trade } from '@/types';
import { format } from 'date-fns';

interface PerformanceChartProps {
  trades: Trade[];
  startingBalance: number;
  balanceHistory?: Array<{ time: string; balance: number }>;
  showDrawdown?: boolean;
}

const PerformanceChart = memo(({ trades, startingBalance, balanceHistory, showDrawdown = false }: PerformanceChartProps) => {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const balanceSeriesRef = useRef<ISeriesApi<'Line'> | null>(null);
  const drawdownSeriesRef = useRef<ISeriesApi<'Area'> | null>(null);

  useEffect(() => {
    if (!chartContainerRef.current) return;

    // Check if dark mode is enabled
    const isDark = document.documentElement.classList.contains('dark');

    // Create chart
    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: isDark ? '#1f2937' : '#ffffff' },
        textColor: isDark ? '#e5e7eb' : '#333',
      },
      width: chartContainerRef.current.clientWidth,
      height: 400,
      grid: {
        vertLines: { color: isDark ? '#374151' : '#f0f0f0' },
        horzLines: { color: isDark ? '#374151' : '#f0f0f0' },
      },
      rightPriceScale: {
        borderColor: isDark ? '#4b5563' : '#e0e0e0',
      },
      timeScale: {
        borderColor: isDark ? '#4b5563' : '#e0e0e0',
        timeVisible: true,
        secondsVisible: false,
      },
    });

    chartRef.current = chart;

    // Create balance line series
    const balanceSeries = chart.addLineSeries({
      color: '#3b82f6',
      lineWidth: 2,
      title: 'Account Balance',
      priceFormat: {
        type: 'price',
        precision: 2,
        minMove: 0.01,
      },
    });
    balanceSeriesRef.current = balanceSeries;

    // Create drawdown area series if requested
    if (showDrawdown) {
      const drawdownSeries = chart.addAreaSeries({
        topColor: 'rgba(239, 68, 68, 0.2)',
        bottomColor: 'rgba(239, 68, 68, 0)',
        lineColor: '#ef4444',
        lineWidth: 2,
        title: 'Drawdown',
        priceFormat: {
          type: 'price',
          precision: 2,
          minMove: 0.01,
        },
      });
      drawdownSeriesRef.current = drawdownSeries;
    }

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current && chartRef.current) {
        chartRef.current.applyOptions({
          width: chartContainerRef.current.clientWidth,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (chartRef.current) {
        chartRef.current.remove();
      }
    };
  }, [showDrawdown]);

  useEffect(() => {
    if (!balanceSeriesRef.current) return;

    let data: LineData[] = [];

    if (balanceHistory && balanceHistory.length > 0) {
      // Use provided balance history
      data = balanceHistory.map(point => ({
        time: point.time,
        value: point.balance,
      }));
    } else if (trades && trades.length > 0) {
      // Calculate balance from trades
      let balance = startingBalance;
      const balancePoints: LineData[] = [
        { time: format(new Date(), 'yyyy-MM-dd'), value: balance },
      ];

      trades
        .filter(trade => trade.net_profit !== undefined)
        .sort((a, b) => new Date(a.open_time).getTime() - new Date(b.open_time).getTime())
        .forEach((trade) => {
          balance += trade.net_profit!;
          const date = format(new Date(trade.close_time || trade.open_time), 'yyyy-MM-dd');
          balancePoints.push({ time: date, value: balance });
        });

      data = balancePoints;
    }

    if (data.length > 0) {
      balanceSeriesRef.current.setData(data);
      chartRef.current?.timeScale().fitContent();
    }

    // Calculate and set drawdown data if enabled
    if (showDrawdown && drawdownSeriesRef.current && data.length > 0) {
      let peak = data[0].value;
      const drawdownData = data.map(point => {
        if (point.value > peak) {
          peak = point.value;
        }
        const drawdown = peak - point.value;
        return {
          time: point.time,
          value: -drawdown, // Negative to show below zero line
        };
      });
      
      drawdownSeriesRef.current.setData(drawdownData);
    }

  }, [trades, startingBalance, balanceHistory, showDrawdown]);

  return (
    <div className="w-full">
      <div ref={chartContainerRef} className="w-full" />
      <div className="mt-4 flex justify-center space-x-6 text-sm text-gray-600 dark:text-gray-400">
        <div className="flex items-center">
          <div className="w-4 h-0.5 bg-blue-500 mr-2"></div>
          <span>Account Balance</span>
        </div>
        {showDrawdown && (
          <div className="flex items-center">
            <div className="w-4 h-0.5 bg-red-500 mr-2"></div>
            <span>Drawdown</span>
          </div>
        )}
      </div>
    </div>
  );
});

PerformanceChart.displayName = 'PerformanceChart';

export default PerformanceChart;