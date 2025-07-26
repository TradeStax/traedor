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

    // Create chart with responsive options
    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: isDark ? '#1f2937' : '#ffffff' },
        textColor: isDark ? '#e5e7eb' : '#333333',
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
    
    // Listen for theme changes
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.type === 'attributes' && mutation.attributeName === 'class') {
          const isDarkNow = document.documentElement.classList.contains('dark');
          if (chartRef.current) {
            chartRef.current.applyOptions({
              layout: {
                background: { type: ColorType.Solid, color: isDarkNow ? '#1f2937' : '#ffffff' },
                textColor: isDarkNow ? '#e5e7eb' : '#333333',
              },
              grid: {
                vertLines: { color: isDarkNow ? '#374151' : '#f0f0f0' },
                horzLines: { color: isDarkNow ? '#374151' : '#f0f0f0' },
              },
              rightPriceScale: {
                borderColor: isDarkNow ? '#4b5563' : '#e0e0e0',
              },
              timeScale: {
                borderColor: isDarkNow ? '#4b5563' : '#e0e0e0',
              },
            });
          }
        }
      });
    });
    
    observer.observe(document.documentElement, { attributes: true });

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
      observer.disconnect();
      if (chartRef.current) {
        chartRef.current.remove();
      }
    };
  }, [showDrawdown]);

  useEffect(() => {
    if (!balanceSeriesRef.current || !chartRef.current) return;

    console.log('PerformanceChart received props:', { 
      tradesLength: trades?.length, 
      startingBalance, 
      balanceHistoryLength: balanceHistory?.length 
    });
    console.log('First few trades:', trades?.slice(0, 3));

    let data: LineData[] = [];

    if (balanceHistory && balanceHistory.length > 0) {
      // Use provided balance history
      data = balanceHistory.map(point => {
        // Convert time string to Unix timestamp (seconds)
        const timeInSeconds = Math.floor(new Date(point.time).getTime() / 1000);
        return {
          time: timeInSeconds as any,
          value: point.balance,
        };
      });
    } else if (trades && trades.length > 0) {
      // Calculate balance from trades
      let balance = startingBalance;
      const balancePoints: LineData[] = [];

      // Helper function to safely parse dates
      const parseTradeDate = (dateStr: string): Date | null => {
        if (!dateStr || dateStr === null || dateStr === undefined) return null;
        const parsed = new Date(dateStr);
        return isNaN(parsed.getTime()) ? null : parsed;
      };

      // Filter and validate trades first
      const today = new Date();
      const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate());
      
      const validTrades = trades.filter(trade => {
        const hasValidProfit = trade.net_profit !== undefined && 
                              trade.net_profit !== null &&
                              !isNaN(trade.net_profit) &&
                              isFinite(trade.net_profit);
        
        const tradeDate = parseTradeDate(trade.open_time);
        const hasValidOpenTime = tradeDate !== null;
        
        // Filter out trades from today (current day) to avoid chart compression
        const isNotToday = tradeDate && tradeDate < todayStart;
        
        if (!hasValidProfit || !hasValidOpenTime || !isNotToday) {
          console.warn('Filtering out invalid/current-day trade:', {
            hasValidProfit,
            hasValidOpenTime,
            isNotToday,
            tradeDate: tradeDate?.toISOString(),
            trade: { 
              net_profit: trade.net_profit, 
              open_time: trade.open_time,
              close_time: trade.close_time 
            }
          });
          return false;
        }
        return true;
      });

      console.log(`Filtered ${trades.length} trades down to ${validTrades.length} valid trades`);

      if (validTrades.length > 0) {
        // Sort by open time
        const sortedTrades = validTrades.sort((a, b) => {
          const timeA = parseTradeDate(a.open_time)!.getTime();
          const timeB = parseTradeDate(b.open_time)!.getTime();
          return timeA - timeB;
        });

        // Add starting point
        const firstTradeTime = parseTradeDate(sortedTrades[0].open_time)!;
        const startTimeInSeconds = Math.floor(firstTradeTime.getTime() / 1000);
        balancePoints.push({ time: startTimeInSeconds as any, value: balance });

        // Process each trade
        sortedTrades.forEach((trade, index) => {
          balance += trade.net_profit!;
          
          // Use close time if available and valid, otherwise use open time
          let tradeEndTime = parseTradeDate(trade.close_time || '');
          if (!tradeEndTime) {
            tradeEndTime = parseTradeDate(trade.open_time)!;
          }
          
          // Convert to Unix timestamp (seconds since epoch) for lightweight-charts
          const timeInSeconds = Math.floor(tradeEndTime.getTime() / 1000);
          
          // Ensure balance is finite
          if (isFinite(balance)) {
            balancePoints.push({ time: timeInSeconds as any, value: balance });
          } else {
            console.error(`Invalid balance calculated at trade ${index}:`, balance, trade);
          }
        });
      }

      data = balancePoints;
    } else {
      // Show a flat line at starting balance if no data
      const todayInSeconds = Math.floor(new Date().getTime() / 1000);
      data = [{ time: todayInSeconds as any, value: startingBalance }];
    }

    // Debug logging and final validation
    console.log('Chart data before rendering:', data);
    console.log('Data length:', data.length);
    
    // Enhanced final validation - remove any invalid data points
    const validateDataPoint = (point: any): point is LineData => {
      if (!point || typeof point !== 'object') return false;
      
      // Validate time - now expecting Unix timestamp (number)
      if (point.time === null || point.time === undefined) return false;
      const numTime = Number(point.time);
      if (isNaN(numTime) || !isFinite(numTime) || numTime <= 0) return false;
      
      // Validate value - must be a finite number
      if (point.value === null || point.value === undefined) return false;
      const numValue = Number(point.value);
      if (isNaN(numValue) || !isFinite(numValue)) return false;
      
      return true;
    };
    
    const validatedData = data.filter(point => {
      const isValid = validateDataPoint(point);
      if (!isValid) {
        console.warn('Removing invalid data point:', point);
      }
      return isValid;
    });
    
    console.log('Chart data after validation:', validatedData);

    if (validatedData.length > 0 && balanceSeriesRef.current) {
      try {
        console.log('Setting chart data:', validatedData);
        
        // Final safety transformation - ensure chart library compatibility
        const chartData = validatedData.map((point, index) => {
          let safeValue = Number(point.value);
          let safeTime = Number(point.time);
          
          // Extra validation - replace any remaining invalid values
          if (!isFinite(safeValue) || isNaN(safeValue)) {
            console.warn(`Fixing invalid value at index ${index}:`, point.value, '-> 0');
            safeValue = 0;
          }
          
          if (!isFinite(safeTime) || isNaN(safeTime) || safeTime <= 0) {
            console.warn(`Fixing invalid time at index ${index}:`, point.time, '-> using previous or current time');
            safeTime = index > 0 ? chartData[index - 1]?.time || Math.floor(Date.now() / 1000) : Math.floor(Date.now() / 1000);
          }
          
          return {
            time: safeTime as any,
            value: safeValue
          };
        }).filter(point => {
          // Final filter to remove any data points that are still invalid
          const isValidTime = isFinite(point.time) && point.time > 0;
          const isValidValue = isFinite(point.value) && !isNaN(point.value);
          
          if (!isValidTime || !isValidValue) {
            console.warn('Removing invalid data point:', point);
            return false;
          }
          return true;
        });
        
        // Sort by time to ensure proper ordering (time is now Unix timestamp)
        chartData.sort((a, b) => a.time - b.time);
        
        // Remove duplicate timestamps (keep the last value for each timestamp)
        const uniqueChartData = chartData.reduce((acc, point) => {
          const existingIndex = acc.findIndex(p => p.time === point.time);
          if (existingIndex >= 0) {
            // Replace existing point with same timestamp
            acc[existingIndex] = point;
          } else {
            acc.push(point);
          }
          return acc;
        }, [] as typeof chartData);
        
        console.log('Final chart data (unique timestamps):', uniqueChartData);
        console.log('Data points count:', uniqueChartData.length);
        
        // Log first few points for debugging
        console.log('First 5 data points:', uniqueChartData.slice(0, 5));
        console.log('Last 5 data points:', uniqueChartData.slice(-5));
        
        // Check time spread
        if (uniqueChartData.length > 1) {
          const firstTime = uniqueChartData[0].time;
          const lastTime = uniqueChartData[uniqueChartData.length - 1].time;
          const timeSpread = lastTime - firstTime;
          const timeSpreadHours = timeSpread / 3600;
          const timeSpreadDays = timeSpread / 86400;
          
          console.log('Time analysis:');
          console.log('  First timestamp:', firstTime, '(', new Date(firstTime * 1000).toISOString(), ')');
          console.log('  Last timestamp:', lastTime, '(', new Date(lastTime * 1000).toISOString(), ')');
          console.log('  Time spread (seconds):', timeSpread);
          console.log('  Time spread (hours):', timeSpreadHours);
          console.log('  Time spread (days):', timeSpreadDays);
          
          if (timeSpread < 3600) {
            console.warn('⚠️ WARNING: Time spread is less than 1 hour - this will create a vertical line!');
          }
          
          // Check for same timestamps
          const allSameTime = uniqueChartData.every(point => point.time === firstTime);
          if (allSameTime) {
            console.error('🚨 ERROR: All data points have the same timestamp!');
          }
        }
        
        // Set chart data with error handling
        try {
          balanceSeriesRef.current.setData(uniqueChartData);
          console.log('Chart data set successfully');
        } catch (chartError) {
          console.error('Error setting chart data:', chartError);
          console.error('Problematic data:', uniqueChartData);
          
          // Try with a minimal safe dataset
          const safeData = [{
            time: Math.floor(Date.now() / 1000) as any,
            value: startingBalance
          }];
          
          try {
            balanceSeriesRef.current.setData(safeData);
            console.log('Used fallback safe data');
          } catch (fallbackError) {
            console.error('Even safe data failed:', fallbackError);
          }
        }
        
        // Use fitContent for better chart scaling
        setTimeout(() => {
          try {
            chartRef.current?.timeScale().fitContent();
          } catch (fitError) {
            console.warn('Error fitting chart content:', fitError);
          }
        }, 100);
      } catch (error) {
        console.error('Error setting chart data:', error);
        console.error('Problem data:', validatedData);
        
        // Show fallback data with proper validation
        const fallbackData = [{ 
          time: Math.floor(new Date().getTime() / 1000) as any, 
          value: Number(startingBalance) || 0 
        }];
        console.log('Using fallback data:', fallbackData);
        try {
          balanceSeriesRef.current.setData(fallbackData);
        } catch (fallbackError) {
          console.error('Even fallback data failed:', fallbackError);
        }
      }
    } else if (validatedData.length === 0) {
      console.warn('No valid data points for chart');
      // Show empty state or fallback
      if (balanceSeriesRef.current) {
        try {
          const fallbackData = [{ 
            time: Math.floor(new Date().getTime() / 1000) as any, 
            value: Number(startingBalance) || 0
          }];
          balanceSeriesRef.current.setData(fallbackData);
        } catch (fallbackError) {
          console.error('Fallback data failed:', fallbackError);
        }
      }
    }

    // Calculate and set drawdown data if enabled
    if (showDrawdown && drawdownSeriesRef.current && validatedData.length > 0) {
      let peak = validatedData[0].value;
      const drawdownData = validatedData.map(point => {
        if (point.value > peak) {
          peak = point.value;
        }
        const drawdown = peak - point.value;
        const drawdownValue = -drawdown; // Negative to show below zero line
        
        // Validate drawdown value
        if (!isFinite(drawdownValue) || isNaN(drawdownValue)) {
          console.warn('Invalid drawdown value:', drawdownValue, 'for point:', point);
          return {
            time: point.time as any,
            value: 0,
          };
        }
        
        return {
          time: point.time as any, // Already in Unix timestamp format
          value: drawdownValue,
        };
      });
      
      // Final validation for drawdown data
      const validDrawdownData = drawdownData.filter(point => 
        validateDataPoint(point)
      );
      
      if (validDrawdownData.length > 0) {
        try {
          drawdownSeriesRef.current.setData(validDrawdownData);
        } catch (error) {
          console.error('Error setting drawdown data:', error);
        }
      }
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