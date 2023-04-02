// Package internal contains the game's core loop.
package internal

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/golang/glog"
)

type keyEvent int

const (
	down keyEvent = iota
	up
	left
	right
)

type position struct {
	x int
	y int
}

type game struct {
	snake []position
	food  map[position]struct{}
}

func newGame() *game {
	return &game{
		snake: []position{{0, 0}},
		food:  map[position]struct{}{{0, 1}: {}},
	}
}

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range text {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func (g *game) drawGame(s tcell.Screen) {
	for _, p := range g.snake {
		s.SetContent(p.x, p.y, 's', nil, tcell.StyleDefault)
	}
	for p := range g.food {
		s.SetContent(p.x, p.y, 'f', nil, tcell.StyleDefault)
	}
}

func (g *game) event(k keyEvent, rows, cols int) bool {
	glog.V(2).Infof("Game: %+v", g)
	// Save the tail so that if the snake ate food we can append the tail to
	// grow by 1 unit.
	tail := g.snake[len(g.snake)-1]
	for i := len(g.snake) - 1; i > 0; i-- {
		g.snake[i] = g.snake[i-1]
	}
	// x, y coordinate system starting from 0, 0 at the upper-left corner.
	switch k {
	case down:
		g.snake[0].y++
	case up:
		g.snake[0].y--
	case left:
		g.snake[0].x--
	case right:
		g.snake[0].x++
	}
	// Range check the snake to end the game if it goes off the screen.
	if g.snake[0].x < 0 || g.snake[0].y < 0 || g.snake[0].x >= rows || g.snake[0].y >= cols {
		return false
	}
	// Check that the snake hasn't run into itself.
	positions := map[position]struct{}{}
	for _, p := range g.snake {
		if _, ok := positions[p]; ok {
			return false
		} else {
			positions[p] = struct{}{}
		}
	}
	// Grow the snake when it eats food.
	if _, ok := g.food[g.snake[0]]; ok {
		glog.V(2).Infof("Ate food: x=%v y=%v", g.snake[0].x, g.snake[0].y)
		g.snake = append(g.snake, tail)
		delete(g.food, g.snake[0])
		x := rand.Intn(rows)
		y := rand.Intn(cols)
		glog.V(2).Infof("New food: x=%v y=%v", x, y)
		g.food[position{x, y}] = struct{}{}
	}
	glog.V(2).Infof("Game: %+v", g)
	return true
}

// Loop starts the game loop. Returns an error if the screen can't be created.
// Cancelling the context will end the loop. Ctrl-C from the user too.
func Loop(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("problem creating screen: %v", err)
	}
	if err := s.Init(); err != nil {
		return fmt.Errorf("init problem: %v", err)
	}
	defer s.Fini()
	g := newGame()
	directions := map[tcell.Key]keyEvent{
		tcell.KeyDown:  down,
		tcell.KeyUp:    up,
		tcell.KeyLeft:  left,
		tcell.KeyRight: right,
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var event *tcell.EventKey
	var mu sync.Mutex
	go func() {
		var finished bool
		for {
			s.Clear()
			if finished {
				// Stop processing key events when the game is over.
				text := "game over"
				drawText(s, 0, 0, len(text), 0, tcell.StyleDefault, text)
			} else {
				g.drawGame(s)
			}
			s.Show()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			mu.Lock()
			e := event
			mu.Unlock()
			if e == nil {
				continue
			}
			d := directions[e.Key()]
			glog.V(2).Infof("Key: %v", d)
			rows, cols := s.Size()
			if !g.event(d, rows, cols) {
				finished = true
			}
		}
	}()
	for {
		glog.Flush()
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		e := s.PollEvent()
		switch e := e.(type) {
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyCtrlC:
				return nil
			case tcell.KeyDown, tcell.KeyUp, tcell.KeyLeft, tcell.KeyRight:
				// A key event updates the current event being drawn propagated
				// forward by the goroutine.
				mu.Lock()
				event = e
				mu.Unlock()
			}
		}
	}
}
