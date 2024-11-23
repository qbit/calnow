package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/sosodev/duration"
)

const (
	fullTimeFmt = "20060102T150405"
	fullFimeTmt = "20060102T150405"
)

var (
	debug = false
)

func msg(fmt string, args ...any) {
	if debug {
		log.Printf(fmt, args...)
	}
}

func between(now, begin, end time.Time) bool {
	return now.After(begin) && now.Before(end)
}

func fixDate(a, b time.Time) time.Time {
	return time.Date(
		a.Year(),
		a.Month(),
		a.Day(),
		a.Hour(),
		a.Minute(),
		a.Second(),
		a.Nanosecond(),
		b.Location(),
	)
}

func main() {
	flag.BoolVar(&debug, "debug", false, "print debug messages")
	flag.Parse()
	username := os.Getenv("CALNOW_USER")
	serverURL := os.Getenv("CALNOW_URL")
	password := os.Getenv("CALNOW_PASS")

	if username == "" || password == "" {
		log.Fatal("Please set CALNOW_USER and CALNOW_PASS environment variables")
	}

	client, err := caldav.NewClient(webdav.HTTPClientWithBasicAuth(
		&http.Client{},
		username,
		password,
	), serverURL)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	princ, err := client.FindCurrentUserPrincipal(ctx)

	homeSets, err := client.FindCalendarHomeSet(ctx, princ)
	if err != nil {
		log.Fatal("Failed to find home set:", err)
	}

	calendars, err := client.FindCalendars(ctx, homeSets)
	if err != nil {
		log.Printf("Failed to find calendars for home set %s: %v", homeSets, err)
	}

	now := time.Now()
	start := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 1, 0,
		now.Location(),
	)

	end := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		23, 59, 59, 999999999,
		now.Location(),
	)

	for _, cal := range calendars {
		if !slices.Contains(cal.SupportedComponentSet, "VEVENT") {
			continue
		}

		path := cal.Path
		query := &caldav.CalendarQuery{
			CompRequest: caldav.CalendarCompRequest{
				Name: "VCALENDAR",
				Comps: []caldav.CalendarCompRequest{{
					Name: "VEVENT",
					Props: []string{
						"SUMMARY",
						"UID",
						"DTSTART",
						"DTEND",
						"DURATION",
					},
				}},
			},

			CompFilter: caldav.CompFilter{
				Name: "VCALENDAR",
				Comps: []caldav.CompFilter{{
					Name:  "VEVENT",
					Start: start,
					End:   end,
				}},
			},
		}
		events, err := client.QueryCalendar(ctx, path, query)
		if err != nil {
			continue
		}
		for _, e := range events {
			for i := range e.Data.Children {
				sum := e.Data.Children[i].Props.Get("SUMMARY")
				end := e.Data.Children[i].Props.Get("DTEND")
				dur := e.Data.Children[i].Props.Get("DURATION")
				begin := e.Data.Children[i].Props.Get("DTSTART")

				if sum != nil {
					msg("%s:%s", cal.Name, sum.Value)
				}

				if begin == nil && dur == nil {
					continue
				}

				if end != nil {
					endTime, err := time.Parse(fullTimeFmt, end.Value)
					if err != nil {
						continue
					}
					beginTime, err := time.Parse(fullTimeFmt, begin.Value)
					if err != nil {
						continue
					}

					endTime = fixDate(endTime, now)
					beginTime = fixDate(beginTime, now)

					msg("%s <%s> %s, between: %t", beginTime, now, endTime, between(now, beginTime, endTime))
					if between(now, beginTime, endTime) {
						os.Exit(0)
					}
				}

				if dur != nil && begin != nil {
					beginTime, err := time.Parse(fullTimeFmt, begin.Value)
					if err != nil {
						continue
					}
					eDur, err := duration.Parse(dur.Value)
					if err != nil {
						continue
					}
					endTime := beginTime.Add(eDur.ToTimeDuration())

					endTime = fixDate(endTime, now)
					beginTime = fixDate(beginTime, now)

					msg("%s <%s> %s, between: %t", beginTime, now, endTime, between(now, beginTime, endTime))
					if between(now, beginTime, endTime) {
						os.Exit(0)
					}
				}
			}
		}
	}

	os.Exit(1)
}
