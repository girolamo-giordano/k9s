package model_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/derailed/k9s/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestFlash(t *testing.T) {
	const delay = 1 * time.Millisecond

	uu := map[string]struct {
		level model.FlashLevel
		e     string
	}{
		"info": {level: model.FlashInfo, e: "blee"},
		"warn": {level: model.FlashWarn, e: "blee"},
		"err":  {level: model.FlashErr, e: "blee"},
	}

	for k := range uu {
		u := uu[k]

		t.Run(k, func(t *testing.T) {
			f := model.NewFlash(delay)
			v := newFlash()
			go v.listen(f.Channel())

			switch u.level {
			case model.FlashInfo:
				f.Info(u.e)
			case model.FlashWarn:
				f.Warn(u.e)
			case model.FlashErr:
				f.Err(errors.New(u.e))
			}

			time.Sleep(5 * delay)
			s, _, l, m := v.getMetrics()
			assert.Equal(t, 1, s)
			assert.Equal(t, u.level, l)
			assert.Equal(t, u.e, m)

			close(f.Channel())
		})
	}
}

func TestFlashBurst(t *testing.T) {
	const delay = 1 * time.Millisecond

	f := model.NewFlash(delay)
	v := newFlash()
	go v.listen(f.Channel())

	count := 5
	for i := 1; i <= count; i++ {
		f.Info(fmt.Sprintf("test-%d", i))
	}

	time.Sleep(2 * delay)
	s, _, l, m := v.getMetrics()
	assert.Equal(t, count, s)
	assert.Equal(t, model.FlashInfo, l)
	assert.Equal(t, fmt.Sprintf("test-%d", count), m)
}

type flash struct {
	set, clear int
	level      model.FlashLevel
	msg        string
	mx         sync.RWMutex
}

func newFlash() *flash {
	return &flash{}
}

func (f *flash) getMetrics() (int, int, model.FlashLevel, string) {
	f.mx.RLock()
	defer f.mx.RUnlock()
	return f.set, f.clear, f.level, f.msg
}

func (f *flash) listen(c model.FlashChan) {
	for m := range c {
		f.mx.Lock()
		{
			if m.IsClear() {
				f.clear++
			} else {
				f.set++
				f.level, f.msg = m.Level, m.Text
			}
		}
		f.mx.Unlock()
	}
}