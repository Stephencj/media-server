package library

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/stephencjuliano/media-server/internal/db"
)

// Keeping the lineup on the clock and the calendar.
//
// A channel's position is counted in elapsed seconds from a fixed moment,
// which knows nothing of clocks going forward or back, of a cycle running
// out, or of December arriving. Left alone, a channel's mornings drift an
// hour twice a year, its fortnight repeats in the same order for ever, and
// its Christmas films never show up. The server runs KeepLineupCurrent every
// few minutes to see to all three.

// refreshHour is when, on the viewer's clock, channels are rebuilt for a new
// cycle or a new season. Rebuilding changes what is on, so it is done once,
// at the hour fewest people are watching.
const refreshHour = 4

// clockDrift is how far a dayparted channel's cycle has slipped from the
// viewer's clock: positive when the clock is ahead of the cycle. It is zero
// until the clocks change, then an hour.
func clockDrift(anchor, now time.Time, location *time.Location) time.Duration {
	local := now.In(location)
	sinceMidnight := local.Hour()*3600 + local.Minute()*60 + local.Second()

	elapsed := int(now.Sub(anchor).Seconds())
	inCycleDay := ((elapsed % daySeconds) + daySeconds) % daySeconds

	drift := sinceMidnight - inCycleDay
	// The shorter way round the day.
	if drift > daySeconds/2 {
		drift -= daySeconds
	} else if drift <= -daySeconds/2 {
		drift += daySeconds
	}
	return time.Duration(drift) * time.Second
}

// KeepLineupCurrent does whatever the moment calls for and returns a line for
// each thing it did. It is safe to call as often as liked.
//
// Clocks: a dayparted channel whose blocks no longer sit at their hours,
// because the clocks changed, is slid back into place. Nothing is
// rescheduled; the channel jumps an hour, as broadcast television does.
//
// Calendar: once a day, after refreshHour, any channel whose cycle has run
// its course is given a fresh one, and any channel with a seasonal source
// that has just come into or gone out of season is rebuilt.
func KeepLineupCurrent(database *db.DB, now time.Time) ([]string, error) {
	var done []string

	states, err := database.ListLineupChannelStates()
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.Timezone == "" {
			continue
		}
		location, err := time.LoadLocation(state.Timezone)
		if err != nil {
			continue
		}
		drift := clockDrift(state.Anchor, now, location)
		if drift > -time.Minute && drift < time.Minute {
			continue
		}
		if err := database.SetChannelAnchor(state.ChannelID, state.Anchor.Add(-drift)); err != nil {
			return done, err
		}
		done = append(done, fmt.Sprintf("%s: moved %s to follow the clock", state.Name, drift))
	}

	raw, refreshedOn, err := database.LoadLineupSpec()
	if err != nil || raw == "" {
		return done, err
	}
	var spec LineupSpec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return done, fmt.Errorf("the stored lineup cannot be read: %w", err)
	}
	location := time.Local
	if spec.Timezone != "" {
		if location, err = time.LoadLocation(spec.Timezone); err != nil {
			return done, err
		}
	}
	local := now.In(location)
	today := local.Format("2006-01-02")
	if local.Hour() < refreshHour || refreshedOn == today {
		return done, nil
	}

	byName := make(map[string]db.LineupChannelState, len(states))
	for _, state := range states {
		byName[state.Name] = state
	}
	due := map[string]bool{}
	for _, channel := range spec.Channels {
		state, managed := byName[channel.Name]
		if !managed {
			continue
		}
		if state.Signature != seasonSignature(channel, local) {
			due[channel.Name] = true
			done = append(done, fmt.Sprintf("%s: rebuilt for the time of year", channel.Name))
			continue
		}
		if state.CycleDays > 0 {
			applied, err := time.ParseInLocation("2006-01-02", state.AppliedOn, location)
			if err == nil && !local.Before(applied.AddDate(0, 0, state.CycleDays)) {
				due[channel.Name] = true
				done = append(done, fmt.Sprintf("%s: new %d-day cycle", channel.Name, state.CycleDays))
			}
		}
	}
	if len(due) > 0 {
		if _, err := ApplyLineupAt(database, spec, true, now, due); err != nil {
			return done, err
		}
	}
	return done, database.MarkLineupRefreshed(today)
}
