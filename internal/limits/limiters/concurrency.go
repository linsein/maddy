/*
Maddy Mail Server - Composable all-in-one email server.
Copyright © 2019-2020 Max Mazurov <fox.cpp@disroot.org>, Maddy Mail Server contributors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package limiters

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Semaphore is a convenience wrapper for a channel that implements
// semaphore-kind synchronization.
//
// If the argument given to the NewSemaphore is negative or zero,
// all methods are no-op.
type Semaphore struct {
	weighted *semaphore.Weighted
}

func NewSemaphore(max int) Semaphore {
	s := Semaphore{weighted: nil}
	if max > 0 {
		s.weighted = semaphore.NewWeighted(int64(max))
	}
	return s
}

func (s Semaphore) TakeContext(ctx context.Context) (ReleaseFunc, error) {
	if s.weighted == nil {
		return func() {}, nil
	}

	err := s.weighted.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			s.weighted.Release(1)
		})
	}, nil
}
