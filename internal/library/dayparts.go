package library

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
)

// Dayparts: a channel that airs different things at different times of day.
//
// A channel is a cycle of programmes played end to end forever, and where it
// is in the cycle is counted from a fixed moment. So a cycle that is exactly a
// whole number of days long, counted from a midnight, puts the same second of
// the cycle at the same time of day on every pass, and "mornings" can simply
// be the stretches of the cycle that fall in the morning.

const daySeconds = 24 * 60 * 60

// daypartBlock is one block of the day and what may air in it. from and to
// are seconds after midnight; to at or before from means it runs through
// midnight.
type daypartBlock struct {
	name string
	from int
	to   int
	pool []db.LineupItem
}

func (b daypartBlock) contains(timeOfDay int) bool {
	if b.from < b.to {
		return timeOfDay >= b.from && timeOfDay < b.to
	}
	return timeOfDay >= b.from || timeOfDay < b.to
}

// remaining is how long the block still has to run at a time of day inside
// it.
func (b daypartBlock) remaining(timeOfDay int) int {
	left := b.to - timeOfDay
	if left <= 0 {
		left += daySeconds
	}
	return left
}

// parseClock reads "06:00" as seconds after midnight. "24:00" is midnight.
func parseClock(text string) (int, error) {
	hours, minutes, ok := strings.Cut(strings.TrimSpace(text), ":")
	h, errH := strconv.Atoi(hours)
	m, errM := strconv.Atoi(minutes)
	if !ok || errH != nil || errM != nil || h < 0 || h > 24 || m < 0 || m > 59 || (h == 24 && m != 0) {
		return 0, fmt.Errorf("%q is not a time of day", text)
	}
	return (h*3600 + m*60) % daySeconds, nil
}

// lookahead is how far down a block's queue the scheduler will reach for a
// programme that fits the time left. Small enough that the order stays close
// to shuffled, large enough to usually find an episode when a film won't fit.
const lookahead = 8

// overrun is how far past the end of its block a programme may run before
// the scheduler would rather pick something shorter.
const overrun = 10 * 60

// buildDaypartCycle lays out days' worth of programmes, each drawn from the
// block that covers the moment it starts. The result runs exactly days*24
// hours, so it can be repeated without the blocks drifting round the clock.
//
// A block ends when the programme running at its end time finishes, so the
// next one starts a few minutes late rather than cutting anything off. Each
// block works through its own shuffled pool and picks up where it left off the
// next day. The one liberty taken is at the very end of the cycle, where the
// last programme's slot is stretched by the few minutes needed to land on
// midnight.
func buildDaypartCycle(blocks []daypartBlock, days int, rng *rand.Rand) ([]db.LineupItem, error) {
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no dayparts")
	}
	// Every minute of the day has to belong to exactly one block.
	for minute := 0; minute < 24*60; minute++ {
		covering := 0
		for _, block := range blocks {
			if block.contains(minute * 60) {
				covering++
			}
		}
		if covering != 1 {
			what := "no daypart covers"
			if covering > 1 {
				what = "more than one daypart covers"
			}
			return nil, fmt.Errorf("%s %02d:%02d", what, minute/60, minute%60)
		}
	}

	queues := make([][]db.LineupItem, len(blocks))
	shortest := make([]int, len(blocks))
	for i, block := range blocks {
		for _, item := range block.pool {
			if item.Duration > 0 {
				queues[i] = append(queues[i], item)
			}
		}
		if len(queues[i]) == 0 {
			return nil, fmt.Errorf("the %s daypart has nothing to air", block.name)
		}
		// Sorted first so that the same pool and seed give the same order.
		sort.SliceStable(queues[i], func(a, b int) bool {
			if queues[i][a].Type != queues[i][b].Type {
				return queues[i][a].Type < queues[i][b].Type
			}
			return queues[i][a].ID < queues[i][b].ID
		})
		rng.Shuffle(len(queues[i]), func(a, b int) { queues[i][a], queues[i][b] = queues[i][b], queues[i][a] })
		shortest[i] = queues[i][0].Duration
		for _, item := range queues[i] {
			if item.Duration < shortest[i] {
				shortest[i] = item.Duration
			}
		}
	}

	// take removes the item at index from a block's queue and sends it to the
	// back, so nothing repeats until the whole pool has aired.
	take := func(block, index int) db.LineupItem {
		queue := queues[block]
		item := queue[index]
		copy(queue[index:], queue[index+1:])
		queue[len(queue)-1] = item
		return item
	}

	total := days * daySeconds
	var cycle []db.LineupItem
	for cursor := 0; cursor < total; {
		timeOfDay := cursor % daySeconds
		block := 0
		for i := range blocks {
			if blocks[i].contains(timeOfDay) {
				block = i
			}
		}
		queue := queues[block]
		untilEnd := total - cursor

		// Closing the cycle: nothing left that fits, so the programme
		// before this gap keeps the slot until midnight.
		if untilEnd < shortest[block] {
			if len(cycle) == 0 {
				return nil, fmt.Errorf("the %s daypart has nothing short enough to air", blocks[block].name)
			}
			cycle[len(cycle)-1].Duration += untilEnd
			break
		}

		window := blocks[block].remaining(timeOfDay) + overrun
		reach := len(queue)
		if untilEnd < window {
			// On the final approach the fit has to be exact, so the whole
			// pool is in play.
			window = untilEnd
		} else if reach > lookahead {
			reach = lookahead
		}

		pick, fallback := -1, 0
		for i := 0; i < reach; i++ {
			if queue[i].Duration <= window {
				pick = i
				break
			}
			if queue[i].Duration < queue[fallback].Duration {
				fallback = i
			}
		}
		if pick < 0 {
			if untilEnd < blocks[block].remaining(timeOfDay)+overrun {
				// Nothing fits before the cycle ends; stretch and stop.
				if len(cycle) == 0 {
					return nil, fmt.Errorf("the %s daypart has nothing short enough to air", blocks[block].name)
				}
				cycle[len(cycle)-1].Duration += untilEnd
				break
			}
			pick = fallback
		}

		item := take(block, pick)
		cycle = append(cycle, item)
		cursor += item.Duration
	}
	return cycle, nil
}
