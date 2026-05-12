package launchd

import "testing"

func TestFormatSchedule(t *testing.T) {
	cases := []struct {
		name      string
		schedule  Schedule
		runAtLoad bool
		want      string
	}{
		{
			name:     "empty",
			schedule: Schedule{},
			want:     "",
		},
		{
			name:      "only RunAtLoad",
			schedule:  Schedule{},
			runAtLoad: true,
			want:      "at load",
		},
		{
			name:     "30 second interval",
			schedule: Schedule{Interval: 30},
			want:     "30s",
		},
		{
			name:     "5 minute interval",
			schedule: Schedule{Interval: 300},
			want:     "5min",
		},
		{
			name:     "1 hour interval",
			schedule: Schedule{Interval: 3600},
			want:     "1h",
		},
		{
			name:     "1h5min interval",
			schedule: Schedule{Interval: 3900},
			want:     "1h5min",
		},
		{
			name:     "1 day interval",
			schedule: Schedule{Interval: 86400},
			want:     "1d",
		},
		{
			name: "single calendar weekday + hour",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: -1, Hour: 9, Day: -1, Weekday: 0, Month: -1},
			}},
			want: "Sun 09:00",
		},
		{
			name: "single calendar Monday morning",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: 30, Hour: 8, Day: -1, Weekday: 1, Month: -1},
			}},
			want: "Mon 08:30",
		},
		{
			name: "single calendar hour only",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: -1, Hour: 9, Day: -1, Weekday: -1, Month: -1},
			}},
			want: "09:00",
		},
		{
			name: "multiple calendar entries",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: -1, Hour: 9, Day: -1, Weekday: 1, Month: -1},
				{Minute: -1, Hour: 17, Day: -1, Weekday: 5, Month: -1},
			}},
			want: "Mon 09:00 +1",
		},
		{
			name: "interval and calendar combined",
			schedule: Schedule{
				Interval: 60,
				Calendar: []CalendarEvent{
					{Minute: -1, Hour: 0, Day: -1, Weekday: 0, Month: -1},
				},
			},
			want: "1min, Sun 00:00",
		},
		{
			name: "RunAtLoad ignored when interval present",
			schedule: Schedule{
				Interval: 300,
			},
			runAtLoad: true,
			want:      "5min",
		},
		{
			name: "weekday 7 is Sunday",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: -1, Hour: 0, Day: -1, Weekday: 7, Month: -1},
			}},
			want: "Sun 00:00",
		},
		{
			name: "day of month",
			schedule: Schedule{Calendar: []CalendarEvent{
				{Minute: 0, Hour: 12, Day: 15, Weekday: -1, Month: -1},
			}},
			want: "day 15 12:00",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSchedule(tc.schedule, tc.runAtLoad)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestClassifyScheduled(t *testing.T) {
	jobWithInterval := Job{Schedule: Schedule{Interval: 300}}
	jobWithCalendar := Job{Schedule: Schedule{Calendar: []CalendarEvent{
		{Minute: -1, Hour: 9, Day: -1, Weekday: 1, Month: -1},
	}}}
	jobRunAtLoad := Job{RunAtLoad: true}
	jobPlain := Job{}

	cases := []struct {
		name string
		job  Job
		in   JobStatus
		want State
	}{
		{
			name: "loaded + interval → scheduled",
			job:  jobWithInterval,
			in:   JobStatus{State: StateLoaded},
			want: StateScheduled,
		},
		{
			name: "stopped + calendar → scheduled",
			job:  jobWithCalendar,
			in:   JobStatus{State: StateStopped},
			want: StateScheduled,
		},
		{
			name: "loaded + RunAtLoad → scheduled",
			job:  jobRunAtLoad,
			in:   JobStatus{State: StateLoaded},
			want: StateScheduled,
		},
		{
			name: "loaded + no schedule → unchanged",
			job:  jobPlain,
			in:   JobStatus{State: StateLoaded},
			want: StateLoaded,
		},
		{
			name: "running stays running even with schedule",
			job:  jobWithInterval,
			in:   JobStatus{State: StateRunning, PID: 123},
			want: StateRunning,
		},
		{
			name: "loaded but has PID → unchanged",
			job:  jobWithInterval,
			in:   JobStatus{State: StateLoaded, PID: 123},
			want: StateLoaded,
		},
		{
			name: "loaded with recent crash → unchanged",
			job:  jobWithInterval,
			in:   JobStatus{State: StateLoaded, LastExitCode: 1},
			want: StateLoaded,
		},
		{
			name: "crashed never becomes scheduled",
			job:  jobWithInterval,
			in:   JobStatus{State: StateCrashed, LastExitCode: 1},
			want: StateCrashed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyScheduled(tc.job, tc.in)
			if got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}
