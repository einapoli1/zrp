import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from "lucide-react";
import { api, type CalendarEvent } from "../lib/api";
import { toast } from "sonner";
const months = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
];

const typeConfig: Record<string, { label: string; color: string; textColor: string; bgColor: string }> = {
  workorder: { label: 'Work Order', color: 'bg-blue-500', textColor: 'text-blue-700', bgColor: 'bg-blue-50' },
  po: { label: 'Purchase Order', color: 'bg-green-500', textColor: 'text-green-700', bgColor: 'bg-green-50' },
  quote: { label: 'Quote', color: 'bg-purple-500', textColor: 'text-purple-700', bgColor: 'bg-purple-50' },
};

function Calendar() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [events, setEvents] = useState<CalendarEvent[]>([]);
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [loading, setLoading] = useState(true);

  const currentYear = currentDate.getFullYear();
  const currentMonth = currentDate.getMonth();

  // Generate calendar grid
  const firstDayOfMonth = new Date(currentYear, currentMonth, 1);
  const lastDayOfMonth = new Date(currentYear, currentMonth + 1, 0);
  const firstDayWeekday = firstDayOfMonth.getDay();
  const daysInMonth = lastDayOfMonth.getDate();

  const calendarDays = [];
  
  // Add empty cells for days before month starts
  for (let i = 0; i < firstDayWeekday; i++) {
    calendarDays.push(null);
  }
  
  // Add days of the month
  for (let day = 1; day <= daysInMonth; day++) {
    calendarDays.push(new Date(currentYear, currentMonth, day));
  }

  useEffect(() => {
    const fetchCalendarData = async () => {
      try {
        setLoading(true);
        const data = await api.getCalendarEvents(currentYear, currentMonth + 1);
        setEvents(data);
      } catch (error) {
        toast.error("Failed to fetch calendar data"); console.error("Failed to fetch calendar data:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchCalendarData();
  }, [currentYear, currentMonth]);

  const navigateMonth = (direction: 'prev' | 'next') => {
    setCurrentDate(prev => {
      const newDate = new Date(prev);
      newDate.setMonth(prev.getMonth() + (direction === 'next' ? 1 : -1));
      return newDate;
    });
    setSelectedDate(null);
  };

  const getEventsForDate = (date: Date) => {
    const dateStr = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
    return events.filter(event => event.date === dateStr);
  };

  const getTypeConfig = (type: string) => {
    return typeConfig[type] || { label: type, color: 'bg-gray-500', textColor: 'text-gray-700', bgColor: 'bg-gray-50' };
  };

  const selectedDateEvents = selectedDate ? getEventsForDate(selectedDate) : [];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading calendar...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Calendar</h1>
        <p className="text-muted-foreground">
          View due dates for work orders, purchase orders, and quotes.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Calendar */}
        <div className="lg:col-span-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
              <CardTitle className="flex items-center gap-2">
                <CalendarIcon className="h-5 w-5" />
                {months[currentMonth]} {currentYear}
              </CardTitle>
              <div className="flex gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigateMonth('prev')}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigateMonth('next')}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {/* Calendar Grid */}
              <div className="grid grid-cols-7 gap-2 mb-4">
                {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
                  <div key={day} className="text-center text-sm font-medium text-muted-foreground py-2">
                    {day}
                  </div>
                ))}
              </div>
              
              <div className="grid grid-cols-7 gap-2">
                {calendarDays.map((date, index) => {
                  if (!date) {
                    return <div key={index} className="aspect-square" />;
                  }
                  
                  const dayEvents = getEventsForDate(date);
                  const isSelected = selectedDate && date.toDateString() === selectedDate.toDateString();
                  const isToday = date.toDateString() === new Date().toDateString();
                  
                  return (
                    <button
                      key={index}
                      onClick={() => setSelectedDate(date)}
                      className={`
                        aspect-square p-2 text-sm border rounded-lg transition-colors
                        hover:bg-accent hover:text-accent-foreground
                        ${isSelected ? 'bg-primary text-primary-foreground' : ''}
                        ${isToday ? 'border-primary' : 'border-border'}
                      `}
                    >
                      <div className="h-full flex flex-col">
                        <span className={`font-medium ${isToday ? 'text-primary' : ''}`}>
                          {date.getDate()}
                        </span>
                        <div className="flex-1 mt-1">
                          {dayEvents.slice(0, 2).map(event => (
                            <div
                              key={event.id}
                              className={`w-2 h-2 rounded-full mb-1 ${getTypeConfig(event.type).color}`}
                            />
                          ))}
                          {dayEvents.length > 2 && (
                            <div className="text-xs text-muted-foreground">
                              +{dayEvents.length - 2}
                            </div>
                          )}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Events Panel */}
        <div>
          <Card>
            <CardHeader>
              <CardTitle>
                {selectedDate 
                  ? `Events for ${selectedDate.toLocaleDateString()}`
                  : 'Select a date'
                }
              </CardTitle>
            </CardHeader>
            <CardContent>
              {selectedDate ? (
                <div className="space-y-3">
                  {selectedDateEvents.length === 0 ? (
                    <p className="text-muted-foreground text-sm">No events on this date</p>
                  ) : (
                    selectedDateEvents.map(event => (
                      <div
                        key={event.id}
                        className={`p-3 rounded-lg ${getTypeConfig(event.type).bgColor}`}
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="flex-1">
                            <div className="font-medium text-sm">{event.title}</div>
                            <Badge 
                              variant="secondary" 
                              className={`mt-1 ${getTypeConfig(event.type).textColor}`}
                            >
                              {getTypeConfig(event.type).label}
                            </Badge>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              ) : (
                <p className="text-muted-foreground text-sm">
                  Click on a date to view events
                </p>
              )}
            </CardContent>
          </Card>

          {/* Legend */}
          <Card className="mt-4">
            <CardHeader>
              <CardTitle className="text-base">Legend</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {Object.entries(typeConfig).map(([_type, config]) => (
                  <div key={config.label} className="flex items-center gap-2">
                    <div className={`w-3 h-3 rounded-full ${config.color}`} />
                    <span className="text-sm">{config.label}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default Calendar;
