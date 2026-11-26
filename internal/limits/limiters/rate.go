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
	"time"

	"golang.org/x/time/rate"
)

// Rate structure implements a basic rate-limiter for requests
//
// If burstSize = 0, all methods are no-op and always succeed.
type Rate struct {
	limiter *rate.Limiter
}

func NewRate(burstSize int, interval time.Duration) Rate {
	r := Rate{limiter: nil}
	if burstSize > 0 {
		r.limiter = rate.NewLimiter(
			// number of events per second
			rate.Limit(float64(burstSize)/interval.Seconds()),
			burstSize)
	}
	return r
}

func (r Rate) TakeContext(ctx context.Context) (ReleaseFunc, error) {
	if r.limiter == nil {
		return func() {}, nil
	}

	if err := r.limiter.Wait(ctx); err != nil {
		// If reqCtx will timeout,
		// r.limiter.Wait will return early with a normal error
		if reqErr := ctx.Err(); reqErr != nil {
			return nil, reqErr
		}
		return nil, context.DeadlineExceeded
	}
	return func() {}, nil
}
