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
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrBucketSetFull = errors.New("limiters: Bucket set is full")

// BucketSet combines a group of Ls into a single key-indexed structure.
// Basically, each unique key gets its own counter. The main use case for
// BucketSet is to apply per-resource rate limiting.
//
// Amount of buckets is limited to a certain value. When the size of internal
// map is around or equal to that value, next TakeContext call will attempt to
// remove any stale buckets from the group. If it is not possible to do so (all
// buckets are in active use), TakeContext will return an ErrBucketSetFull.
//
// A BucksetSet without a New function assigned is no-op: Take and TakeContext
// always succeed and Release does nothing.
type BucketSet struct {
	// New function is used to construct underlying L instances.
	//
	// It is safe to change it only when BucketSet is not used by any
	// goroutine.
	New func() L

	// Time after which bucket is considered stale and can be removed from the
	// set. For safe use with Rate limiter, it should be at least as twice as
	// big as Rate refill interval.
	ReapInterval time.Duration

	MaxBuckets int

	mLck sync.Mutex
	m    map[string]*bucketItem
}

type bucketItem struct {
	r        L
	refCount atomic.Int64
	lastUse  time.Time
}

func NewBucketSet(new_ func() L, reapInterval time.Duration, maxBuckets int) *BucketSet {
	return &BucketSet{
		New:          new_,
		ReapInterval: reapInterval,
		MaxBuckets:   maxBuckets,
		m:            map[string]*bucketItem{},
	}
}

func (r *BucketSet) take(key string) *bucketItem {
	r.mLck.Lock()
	defer r.mLck.Unlock()
	bucket, ok := r.m[key]

	// Reap stale buckets when need create new bucket
	if !ok && len(r.m) > r.MaxBuckets {
		now := time.Now()
		// Attempt to get rid of stale buckets.
		for k, v := range r.m {
			if now.Sub(v.lastUse) > r.ReapInterval && v.refCount.Load() == 0 {
				delete(r.m, k)
			}
		}

		// Still full? E.g. all buckets are in use.
		if len(r.m) > r.MaxBuckets {
			return nil
		}
	}

	if !ok {
		r.m[key] = &bucketItem{
			r:        r.New(),
			refCount: atomic.Int64{},
			lastUse:  time.Now(),
		}
		bucket = r.m[key]
	}
	r.m[key].lastUse = time.Now()
	bucket.refCount.Add(1)

	return bucket
}

func (r *BucketSet) TakeContext(ctx context.Context, key string) (ReleaseFunc, error) {
	if r.New == nil {
		return func() {}, nil
	}

	bucket := r.take(key)
	if bucket == nil {
		return nil, ErrBucketSetFull
	}

	release, err := bucket.r.TakeContext(ctx)
	if err != nil {
		bucket.refCount.Add(-1)
		return nil, err
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			release()
			bucket.refCount.Add(-1)
		})
	}, nil
}
